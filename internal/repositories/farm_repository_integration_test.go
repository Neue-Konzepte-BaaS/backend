package repositories_test

import (
	"context"
	"errors"
	"testing"

	"github.com/Neue-Konzepte-BaaS/backend/internal/models"
	"github.com/Neue-Konzepte-BaaS/backend/internal/repositories"
	database "github.com/Neue-Konzepte-BaaS/backend/internal/repositories/db"
	"github.com/Neue-Konzepte-BaaS/backend/internal/services"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

func TestFarmRepository_GetFarmByID(t *testing.T) {
	pool := setupTestDB(t)
	ctx := context.Background()

	farmRepo := repositories.NewFarmRepository(database.New(pool))

	t.Run("farmer with no fields or plots gets zero total, not an error", func(t *testing.T) {
		farmer, farmID, _, _ := seedFarmWithPlots(t, ctx, pool, 0)

		farm, err := farmRepo.GetFarmByID(ctx, farmID)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if farm.ID != farmID {
			t.Errorf("farm id = %v, want %v", farm.ID, farmID)
		}
		if farm.FarmerID != farmer {
			t.Errorf("farmer id = %v, want %v", farm.FarmerID, farmer)
		}
		if farm.Name != "Green Acres" {
			t.Errorf("name = %q, want Green Acres", farm.Name)
		}
		if farm.Address != "1 Farm Lane" {
			t.Errorf("address = %q, want '1 Farm Lane'", farm.Address)
		}
		if farm.TotalSquareMeters != 0 {
			t.Errorf("total square meters = %v, want 0", farm.TotalSquareMeters)
		}
		if farm.FoundedAt != nil {
			t.Errorf("founded at = %v, want nil", farm.FoundedAt)
		}
	})

	t.Run("total square meters sums plots across fields", func(t *testing.T) {
		_, farmID, _, _ := seedFarmWithPlots(t, ctx, pool, 3)

		farm, err := farmRepo.GetFarmByID(ctx, farmID)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if farm.TotalSquareMeters <= 0 {
			t.Errorf("total square meters = %v, want > 0", farm.TotalSquareMeters)
		}
	})

	t.Run("unknown farm id returns ErrNotFound", func(t *testing.T) {
		_, err := farmRepo.GetFarmByID(ctx, uuid.New())
		if err == nil {
			t.Fatal("expected an error for an unknown farm id")
		}
		if !errors.Is(err, services.ErrNotFound) {
			t.Errorf("err = %v, want ErrNotFound", err)
		}
	})
}

func TestFarmRepository_GetFarmIDByFarmerID(t *testing.T) {
	pool := setupTestDB(t)
	ctx := context.Background()

	farmRepo := repositories.NewFarmRepository(database.New(pool))

	t.Run("resolves the farmer's own farm id", func(t *testing.T) {
		farmer, farmID, _, _ := seedFarmWithPlots(t, ctx, pool, 0)

		got, err := farmRepo.GetFarmIDByFarmerID(ctx, farmer)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got != farmID {
			t.Errorf("farm id = %v, want %v", got, farmID)
		}
	})

	t.Run("unknown farmer id returns ErrNotFound", func(t *testing.T) {
		_, err := farmRepo.GetFarmIDByFarmerID(ctx, uuid.New())
		if err == nil {
			t.Fatal("expected an error for an unknown farmer id")
		}
		if !errors.Is(err, services.ErrNotFound) {
			t.Errorf("err = %v, want ErrNotFound", err)
		}
	})
}

// seedFarmWithoutFields creates a farm that owns nothing at all -- the case the
// LEFT JOIN LATERALs in ListFarms exist for. seedFarmWithPlots(…, 0) is not the
// same thing: that farm still has a field.
func seedFarmWithoutFields(t *testing.T, ctx context.Context, pool *pgxpool.Pool, farmName string, postalCode int32) uuid.UUID {
	t.Helper()

	queries := database.New(pool)
	accountRepo := repositories.NewAccountRepository(pool, queries)
	farmRepo := repositories.NewFarmRepository(queries)

	farmer, err := accountRepo.CreateFarmer(ctx, models.Account{
		FirstName:    "Bare",
		LastName:     "Acres",
		Email:        uuid.NewString() + "@example.com",
		PasswordHash: "irrelevant",
	}, farmName, postalCode, "2 Empty Road", "")
	if err != nil {
		t.Fatalf("creating farmer: %v", err)
	}

	farmID, err := farmRepo.GetFarmIDByFarmerID(ctx, farmer.ID)
	if err != nil {
		t.Fatalf("looking up farm: %v", err)
	}
	return farmID
}

// expireRental backdates a rental so its period no longer covers now. The
// repository can only create rentals starting at the current instant, so a
// historic one has to be written directly.
func expireRental(t *testing.T, ctx context.Context, pool *pgxpool.Pool, rental uuid.UUID) {
	t.Helper()

	_, err := pool.Exec(ctx, `
		UPDATE rental
		SET period = tstzrange(CURRENT_TIMESTAMP - interval '60 days', CURRENT_TIMESTAMP - interval '30 days')
		WHERE id = $1`, rental)
	if err != nil {
		t.Fatalf("expiring rental: %v", err)
	}
}

func listAllFarms(t *testing.T, ctx context.Context, repo services.FarmRepository) models.Page[models.FarmListing] {
	t.Helper()

	page, err := repo.ListFarms(ctx, models.FarmListFilter{Limit: 100})
	if err != nil {
		t.Fatalf("listing farms: %v", err)
	}
	return page
}

func findFarm(t *testing.T, page models.Page[models.FarmListing], farmID uuid.UUID) models.FarmListing {
	t.Helper()

	for _, farm := range page.Items {
		if farm.ID == farmID {
			return farm
		}
	}
	t.Fatalf("farm %v is not in the listing", farmID)
	return models.FarmListing{}
}

// A farm that owns nothing must still be listed, as a row of zeros. Getting
// this wrong -- by aggregating through an inner join -- makes every new farm
// invisible to the admin until it creates its first field.
func TestListFarms_IncludesAFarmThatOwnsNothing(t *testing.T) {
	pool := setupTestDB(t)
	ctx := context.Background()
	repo := repositories.NewFarmRepository(database.New(pool))

	bare := seedFarmWithoutFields(t, ctx, pool, "Aardvark Farm", 10115)

	farm := findFarm(t, listAllFarms(t, ctx, repo), bare)

	if farm.Name != "Aardvark Farm" || farm.PostalCode != 10115 {
		t.Errorf("farm = %q/%d, want %q/%d", farm.Name, farm.PostalCode, "Aardvark Farm", 10115)
	}
	if farm.Address != "2 Empty Road" {
		t.Errorf("address = %q, want %q", farm.Address, "2 Empty Road")
	}
	if farm.Fields.Total != 0 || farm.Fields.AreaSquareMeters != 0 {
		t.Errorf("fields = %+v, want zeros", farm.Fields)
	}
	if farm.Plots.Total != 0 || farm.Plots.Rented != 0 || farm.Plots.AreaSquareMeters != 0 {
		t.Errorf("plots = %+v, want zeros", farm.Plots)
	}
	if farm.ActiveRentals != 0 {
		t.Errorf("activeRentals = %d, want 0", farm.ActiveRentals)
	}
}

// The admin farm list and the platform statistics are two views of the same
// figures. If they ever disagree, one of the two queries has drifted -- this is
// the test that says so.
func TestListFarms_ReconcilesWithPlatformStatistics(t *testing.T) {
	pool := setupTestDB(t)
	ctx := context.Background()
	queries := database.New(pool)
	farmRepo := repositories.NewFarmRepository(queries)
	statisticsRepo := repositories.NewStatisticsRepository(queries)
	rentalRepo := repositories.NewRentalRepository(queries)

	// Three farms with different shapes, so the sums are not trivially equal:
	// one with rentals, one with plots but none rented, one with nothing.
	_, busyFarmID, busyPlots, busyCrop := seedFarmWithPlots(t, ctx, pool, 3)
	seedFarmWithPlots(t, ctx, pool, 2)
	seedFarmWithoutFields(t, ctx, pool, "Aardvark Farm", 10115)

	customer := seedCustomer(t, ctx, pool)
	if _, err := rentalRepo.CreateRental(ctx, busyPlots[0], customer, busyCrop, 6); err != nil {
		t.Fatalf("renting a plot: %v", err)
	}

	page := listAllFarms(t, ctx, farmRepo)
	platform, err := statisticsRepo.GetPlatformStatistics(ctx)
	if err != nil {
		t.Fatalf("getting platform statistics: %v", err)
	}

	var fields, plots, rented int64
	for _, farm := range page.Items {
		fields += farm.Fields.Total
		plots += farm.Plots.Total
		rented += farm.Plots.Rented
	}

	if fields != platform.Fields.Total {
		t.Errorf("summed field count = %d, platform statistics says %d", fields, platform.Fields.Total)
	}
	if plots != platform.Plots.Total {
		t.Errorf("summed plot count = %d, platform statistics says %d", plots, platform.Plots.Total)
	}
	if rented != platform.Plots.Rented {
		t.Errorf("summed rented count = %d, platform statistics says %d", rented, platform.Plots.Rented)
	}

	// And the busy farm is actually busy, so the equality above is not three
	// zeros agreeing with three zeros.
	busyFarm := findFarm(t, page, busyFarmID)
	if busyFarm.Plots.Rented != 1 || busyFarm.ActiveRentals != 1 {
		t.Errorf("busy farm rented = %d, activeRentals = %d, want 1 and 1", busyFarm.Plots.Rented, busyFarm.ActiveRentals)
	}
}

// A rental that has run out stops counting, with no cleanup job -- the same
// promise the statistics and plot-search queries make.
func TestListFarms_ExpiredRentalsStopCounting(t *testing.T) {
	pool := setupTestDB(t)
	ctx := context.Background()
	queries := database.New(pool)
	farmRepo := repositories.NewFarmRepository(queries)
	rentalRepo := repositories.NewRentalRepository(queries)

	_, farmID, plotIDs, crop := seedFarmWithPlots(t, ctx, pool, 2)
	customer := seedCustomer(t, ctx, pool)

	live, err := rentalRepo.CreateRental(ctx, plotIDs[0], customer, crop, 6)
	if err != nil {
		t.Fatalf("renting the first plot: %v", err)
	}
	expired, err := rentalRepo.CreateRental(ctx, plotIDs[1], customer, crop, 6)
	if err != nil {
		t.Fatalf("renting the second plot: %v", err)
	}
	expireRental(t, ctx, pool, expired.ID)

	farm := findFarm(t, listAllFarms(t, ctx, farmRepo), farmID)

	if farm.Plots.Total != 2 {
		t.Errorf("plot count = %d, want 2", farm.Plots.Total)
	}
	if farm.Plots.Rented != 1 {
		t.Errorf("rented = %d, want 1: only %v is still live", farm.Plots.Rented, live.ID)
	}
	if farm.ActiveRentals != 1 {
		t.Errorf("activeRentals = %d, want 1", farm.ActiveRentals)
	}
}

// Total is the number of matching rows, not the number on this page, and
// paging through must show every farm exactly once.
func TestListFarms_PagesStablyAndReportsTheFullTotal(t *testing.T) {
	pool := setupTestDB(t)
	ctx := context.Background()
	repo := repositories.NewFarmRepository(database.New(pool))

	for _, name := range []string{"Farm A", "Farm B", "Farm C"} {
		seedFarmWithoutFields(t, ctx, pool, name, 10115)
	}

	seen := map[uuid.UUID]int{}
	for offset := int32(0); offset < 4; offset += 2 {
		page, err := repo.ListFarms(ctx, models.FarmListFilter{Limit: 2, Offset: offset})
		if err != nil {
			t.Fatalf("listing farms at offset %d: %v", offset, err)
		}
		if len(page.Items) > 0 && page.Total != 3 {
			t.Errorf("total at offset %d = %d, want 3", offset, page.Total)
		}
		for _, farm := range page.Items {
			seen[farm.ID]++
		}
	}

	if len(seen) != 3 {
		t.Errorf("saw %d distinct farms across the pages, want 3", len(seen))
	}
	for farmID, count := range seen {
		if count != 1 {
			t.Errorf("farm %v appeared %d times across the pages, want once", farmID, count)
		}
	}
}

func TestListFarms_FiltersNarrowBothItemsAndTotal(t *testing.T) {
	pool := setupTestDB(t)
	ctx := context.Background()
	repo := repositories.NewFarmRepository(database.New(pool))

	seedFarmWithoutFields(t, ctx, pool, "Green Acres", 76133)
	seedFarmWithoutFields(t, ctx, pool, "Aardvark Farm", 10115)
	seedFarmWithoutFields(t, ctx, pool, "Greenfield Hof", 10115)

	berlin := int32(10115)
	tests := []struct {
		name   string
		filter models.FarmListFilter
		want   int64
	}{
		{name: "no filter", filter: models.FarmListFilter{Limit: 100}, want: 3},
		{name: "by name", filter: models.FarmListFilter{Query: "green", Limit: 100}, want: 2},
		{name: "by postal code", filter: models.FarmListFilter{PostalCode: &berlin, Limit: 100}, want: 2},
		{name: "both together", filter: models.FarmListFilter{Query: "green", PostalCode: &berlin, Limit: 100}, want: 1},
		{name: "matching nothing", filter: models.FarmListFilter{Query: "nonesuch", Limit: 100}, want: 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			page, err := repo.ListFarms(ctx, tt.filter)
			if err != nil {
				t.Fatalf("listing farms: %v", err)
			}
			if int64(len(page.Items)) != tt.want {
				t.Errorf("items = %d, want %d", len(page.Items), tt.want)
			}
			// A filter matching nothing returns no rows and therefore no
			// window-function count either.
			if tt.want > 0 && page.Total != tt.want {
				t.Errorf("total = %d, want %d", page.Total, tt.want)
			}
		})
	}
}
