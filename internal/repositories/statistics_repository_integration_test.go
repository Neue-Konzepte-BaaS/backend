package repositories_test

import (
	"context"
	"testing"
	"time"

	"github.com/Neue-Konzepte-BaaS/backend/internal/models"
	"github.com/Neue-Konzepte-BaaS/backend/internal/repositories"
	database "github.com/Neue-Konzepte-BaaS/backend/internal/repositories/db"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// seedFarmWithPlots creates a farmer with one farm and one field containing
// plotCount plots (each a distinct rectangle, so their areas don't overlap),
// offers a crop on each of those plots, and returns the farmer id, the farm
// id, the created plot ids, and the crop id.
func seedFarmWithPlots(t *testing.T, ctx context.Context, pool *pgxpool.Pool, plotCount int) (uuid.UUID, uuid.UUID, []uuid.UUID, uuid.UUID) {
	t.Helper()

	queries := database.New(pool)
	accountRepo := repositories.NewAccountRepository(pool, queries)
	farmRepo := repositories.NewFarmRepository(pool, queries)
	fieldRepo := repositories.NewFieldRepository(queries)
	plotRepo := repositories.NewPlotRepository(queries)
	cropRepo := repositories.NewCropRepository(pool, queries)

	farmer, err := accountRepo.CreateFarmer(ctx, models.Account{
		FirstName:    "Old",
		LastName:     "MacDonald",
		Email:        uuid.NewString() + "@example.com",
		PasswordHash: "irrelevant",
	}, "Green Acres", 76133, "1 Farm Lane", "A small family farm")
	if err != nil {
		t.Fatalf("creating farmer: %v", err)
	}

	farmID, err := farmRepo.GetFarmIDByFarmerID(ctx, farmer.ID)
	if err != nil {
		t.Fatalf("looking up farm: %v", err)
	}

	fieldID, err := fieldRepo.CreateField(ctx, models.Field{
		Name:        "Field 1",
		Farm:        farmID,
		Coordinates: rectangle(0, 0, 100, 100),
	})
	if err != nil {
		t.Fatalf("creating field: %v", err)
	}

	cropName := "Tomatoes-" + uuid.NewString()
	crop, err := cropRepo.CreateCrop(ctx, cropName, cropName, 6)
	if err != nil {
		t.Fatalf("creating crop: %v", err)
	}

	plotIDs := make([]uuid.UUID, plotCount)
	for i := range plotCount {
		minX := float64(i * 10)
		plot, err := plotRepo.CreatePlot(ctx, models.Plot{
			Name:        "Plot",
			Field:       fieldID,
			Coordinates: rectangle(minX, 0, minX+5, 5),
		})
		if err != nil {
			t.Fatalf("creating plot %d: %v", i, err)
		}
		if err := cropRepo.SetPlotCrops(ctx, plot.ID, 100, []uuid.UUID{crop.ID}); err != nil {
			t.Fatalf("offering crop on plot %d: %v", i, err)
		}
		plotIDs[i] = plot.ID
	}

	return farmer.ID, farmID, plotIDs, crop.ID
}

// seedCheckout records a rental_checkout in the given status, written
// directly like rentNow since the full Stripe webhook flow is not what these
// tests are exercising -- only the resulting revenue figures are. rental may
// be uuid.Nil for a status that never produced one (e.g. "pending").
func seedCheckout(t *testing.T, ctx context.Context, pool *pgxpool.Pool, plot, customer, crop uuid.UUID, amountCents int32, status string, rental uuid.UUID) {
	t.Helper()

	var rentalArg any
	if rental != uuid.Nil {
		rentalArg = rental
	}
	_, err := pool.Exec(ctx, `
		INSERT INTO rental_checkout (customer, plot, crop, start_at, message, stripe_checkout_session_id, status, amount_cents, rental)
		VALUES ($1, $2, $3, CURRENT_TIMESTAMP, 'please', $4, $5, $6, $7)`,
		customer, plot, crop, uuid.NewString(), status, amountCents, rentalArg,
	)
	if err != nil {
		t.Fatalf("inserting %s rental checkout: %v", status, err)
	}
}

// TestStatisticsRepository_FarmAndPlatformScope walks through the scenarios
// that most exercise the statistics queries against a real database, sharing
// one container across sub-cases since each container start is expensive.
func TestStatisticsRepository_FarmAndPlatformScope(t *testing.T) {
	pool := setupTestDB(t)
	ctx := context.Background()

	statsRepo := repositories.NewStatisticsRepository(database.New(pool))

	t.Run("fresh farmer with nothing gets all zeros, not an error", func(t *testing.T) {
		_, farmID, _, _ := seedFarmWithPlots(t, ctx, pool, 0)

		stats, err := statsRepo.GetFarmStatistics(ctx, farmID)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if stats.Fields.Total != 1 {
			t.Errorf("fields total = %d, want 1 (the field itself, with no plots)", stats.Fields.Total)
		}
		if stats.Plots.Total != 0 || stats.Plots.Rented != 0 {
			t.Errorf("plots = %+v, want all zero", stats.Plots)
		}
		if stats.Plots.AreaSquareMeters != 0 {
			t.Errorf("plot area = %v, want 0", stats.Plots.AreaSquareMeters)
		}
		if stats.Rentals.Total != 0 || stats.Rentals.Active != 0 || stats.Rentals.Last30Days != 0 {
			t.Errorf("rentals = %+v, want all zero", stats.Rentals)
		}
		if stats.Revenue.TotalCents != 0 || stats.Revenue.Last30DaysCents != 0 {
			t.Errorf("revenue = %+v, want all zero", stats.Revenue)
		}
		if stats.GeneratedAt.IsZero() {
			t.Error("expected GeneratedAt to be set from the database clock")
		}
	})

	t.Run("counts and areas for a farmer with plots, one rented", func(t *testing.T) {
		_, farmID, plots, crop := seedFarmWithPlots(t, ctx, pool, 2)
		customer := seedCustomer(t, ctx, pool)

		rental := rentNow(t, ctx, pool, plots[0], customer, crop, 6)
		seedCheckout(t, ctx, pool, plots[0], customer, crop, 5000, "completed", rental.ID)
		// A still-pending checkout on the other plot must not count as revenue.
		seedCheckout(t, ctx, pool, plots[1], customer, crop, 7500, "pending", uuid.Nil)

		stats, err := statsRepo.GetFarmStatistics(ctx, farmID)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if stats.Fields.Total != 1 {
			t.Errorf("fields total = %d, want 1", stats.Fields.Total)
		}
		if stats.Fields.AreaSquareMeters <= 0 {
			t.Errorf("field area = %v, want > 0", stats.Fields.AreaSquareMeters)
		}
		if stats.Plots.Total != 2 {
			t.Errorf("plots total = %d, want 2", stats.Plots.Total)
		}
		if stats.Plots.Rented != 1 {
			t.Errorf("plots rented = %d, want 1", stats.Plots.Rented)
		}
		if stats.Plots.AreaSquareMeters <= 0 {
			t.Errorf("plot area = %v, want > 0", stats.Plots.AreaSquareMeters)
		}
		if stats.Rentals.Total != 1 || stats.Rentals.Active != 1 || stats.Rentals.Last30Days != 1 {
			t.Errorf("rentals = %+v, want {total:1 active:1 last30Days:1}", stats.Rentals)
		}
		if stats.Revenue.TotalCents != 5000 {
			t.Errorf("revenue total = %d, want 5000 (the pending checkout must not count)", stats.Revenue.TotalCents)
		}
		if stats.Revenue.Last30DaysCents != 5000 {
			t.Errorf("revenue last 30 days = %d, want 5000", stats.Revenue.Last30DaysCents)
		}
	})

	t.Run("isolation: a second farmer's data does not affect the first", func(t *testing.T) {
		_, firstFarmID, _, _ := seedFarmWithPlots(t, ctx, pool, 3)

		before, err := statsRepo.GetFarmStatistics(ctx, firstFarmID)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		// A second, unrelated farmer with its own field, plots and rental.
		_, secondFarmID, secondPlots, secondCrop := seedFarmWithPlots(t, ctx, pool, 5)
		secondCustomer := seedCustomer(t, ctx, pool)
		rentNow(t, ctx, pool, secondPlots[0], secondCustomer, secondCrop, 6)

		after, err := statsRepo.GetFarmStatistics(ctx, firstFarmID)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		// GeneratedAt legitimately differs between the two calls (it is read
		// from the database clock), so it is excluded from this comparison.
		before.GeneratedAt, after.GeneratedAt = time.Time{}, time.Time{}
		if after != before {
			t.Errorf("first farmer's statistics changed after seeding a second farmer: before=%+v after=%+v", before, after)
		}

		// Sanity check the second farmer really did get their own numbers,
		// so a bug that returns the SAME farmer's data for both ids doesn't
		// pass this test by accident.
		secondStats, err := statsRepo.GetFarmStatistics(ctx, secondFarmID)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if secondStats.Plots.Total != 5 {
			t.Errorf("second farmer plots total = %d, want 5", secondStats.Plots.Total)
		}
		if secondStats.Rentals.Active != 1 {
			t.Errorf("second farmer active rentals = %d, want 1", secondStats.Rentals.Active)
		}
	})

	t.Run("platform totals cover every farmer and fill accounts", func(t *testing.T) {
		platformBefore, err := statsRepo.GetPlatformStatistics(ctx)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		_, farmID, plots, crop := seedFarmWithPlots(t, ctx, pool, 1)
		customer := seedCustomer(t, ctx, pool)
		rental := rentNow(t, ctx, pool, plots[0], customer, crop, 6)
		seedCheckout(t, ctx, pool, plots[0], customer, crop, 12345, "completed", rental.ID)

		platformAfter, err := statsRepo.GetPlatformStatistics(ctx)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if platformAfter.Fields.Total != platformBefore.Fields.Total+1 {
			t.Errorf("platform fields total = %d, want %d", platformAfter.Fields.Total, platformBefore.Fields.Total+1)
		}
		if platformAfter.Plots.Total != platformBefore.Plots.Total+1 {
			t.Errorf("platform plots total = %d, want %d", platformAfter.Plots.Total, platformBefore.Plots.Total+1)
		}
		if platformAfter.Rentals.Active != platformBefore.Rentals.Active+1 {
			t.Errorf("platform active rentals = %d, want %d", platformAfter.Rentals.Active, platformBefore.Rentals.Active+1)
		}
		if platformAfter.Revenue.TotalCents != platformBefore.Revenue.TotalCents+12345 {
			t.Errorf("platform revenue total = %d, want %d", platformAfter.Revenue.TotalCents, platformBefore.Revenue.TotalCents+12345)
		}
		if platformAfter.Revenue.Last30DaysCents != platformBefore.Revenue.Last30DaysCents+12345 {
			t.Errorf("platform revenue last 30 days = %d, want %d", platformAfter.Revenue.Last30DaysCents, platformBefore.Revenue.Last30DaysCents+12345)
		}

		if platformAfter.Accounts == nil {
			t.Fatal("platform statistics must include the accounts group")
		}
		if platformAfter.Accounts.Farmers != platformBefore.Accounts.Farmers+1 {
			t.Errorf("platform farmers = %d, want %d", platformAfter.Accounts.Farmers, platformBefore.Accounts.Farmers+1)
		}
		if platformAfter.Accounts.Customers != platformBefore.Accounts.Customers+1 {
			t.Errorf("platform customers = %d, want %d", platformAfter.Accounts.Customers, platformBefore.Accounts.Customers+1)
		}

		// Consistency: the farm-scoped view of the farmer just created must
		// agree with what went into the platform totals above.
		farmStats, err := statsRepo.GetFarmStatistics(ctx, farmID)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if farmStats.Plots.Total != 1 || farmStats.Rentals.Active != 1 || farmStats.Revenue.TotalCents != 12345 {
			t.Errorf("farm stats = %+v, want plots.total=1 rentals.active=1 revenue.totalCents=12345", farmStats)
		}
	})
}
