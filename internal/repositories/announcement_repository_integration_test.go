package repositories_test

import (
	"context"
	"testing"

	"github.com/Neue-Konzepte-BaaS/backend/internal/models"
	"github.com/Neue-Konzepte-BaaS/backend/internal/repositories"
	database "github.com/Neue-Konzepte-BaaS/backend/internal/repositories/db"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// seedFarmerWithPlots creates a farmer owning one field with plotCount plots,
// each offering one crop, and returns the farmer, the plots and that crop.
func seedFarmerWithPlots(t *testing.T, ctx context.Context, pool *pgxpool.Pool, plotCount int) (uuid.UUID, []uuid.UUID, uuid.UUID) {
	t.Helper()

	queries := database.New(pool)
	accountRepo := repositories.NewAccountRepository(pool, queries)
	fieldRepo := repositories.NewFieldRepository(queries)
	plotRepo := repositories.NewPlotRepository(queries)
	cropRepo := repositories.NewCropRepository(pool, queries)

	farmer, err := accountRepo.CreateFarmer(ctx, models.Account{
		FirstName:    "Old",
		LastName:     "MacDonald",
		Email:        uuid.NewString() + "@example.com",
		PasswordHash: "irrelevant",
	}, "Green Acres", 76133)
	if err != nil {
		t.Fatalf("creating farmer: %v", err)
	}

	fieldID, err := fieldRepo.CreateField(ctx, models.Field{
		Name:        "Field 1",
		Farmer:      farmer.ID,
		Coordinates: rectangle(0, 0, 100, 100),
	})
	if err != nil {
		t.Fatalf("creating field: %v", err)
	}

	crop, err := cropRepo.CreateCrop(ctx, "Tomatoes-"+uuid.NewString(), 6)
	if err != nil {
		t.Fatalf("creating crop: %v", err)
	}

	plots := make([]uuid.UUID, plotCount)
	for i := range plots {
		// Side by side inside the field, so no two plots overlap.
		minX := float64(i*10 + 1)
		plot, err := plotRepo.CreatePlot(ctx, models.Plot{
			Name:        "Plot",
			Field:       fieldID,
			Coordinates: rectangle(minX, 1, minX+5, 5),
		})
		if err != nil {
			t.Fatalf("creating plot %d: %v", i, err)
		}
		if err := cropRepo.SetPlotCrops(ctx, plot.ID, []uuid.UUID{crop.ID}); err != nil {
			t.Fatalf("offering crop on plot %d: %v", i, err)
		}
		plots[i] = plot.ID
	}

	return farmer.ID, plots, crop.ID
}

// rentPast books a plot for a period that has already ended. The repository
// always starts a rental at CURRENT_TIMESTAMP, so an expired one has to be
// written directly.
func rentPast(t *testing.T, ctx context.Context, pool *pgxpool.Pool, plot, customer, crop uuid.UUID) {
	t.Helper()

	_, err := pool.Exec(ctx, `
		INSERT INTO rental (plot, customer, crop, period)
		VALUES ($1, $2, $3, tstzrange(CURRENT_TIMESTAMP - interval '2 months', CURRENT_TIMESTAMP - interval '1 month'))`,
		plot, customer, crop)
	if err != nil {
		t.Fatalf("inserting expired rental: %v", err)
	}
}

// TestGetCustomersOfFarmer_CountsEachCustomerOnce is the test the whole
// announcement fan-out rests on: the query joins account through rental, plot
// and field, so a customer renting several plots from the same farmer appears
// once per rental unless the query deduplicates. Mailing that customer three
// times for one announcement is the failure this guards against.
func TestGetCustomersOfFarmer_CountsEachCustomerOnce(t *testing.T) {
	pool := setupTestDB(t)
	ctx := context.Background()

	queries := database.New(pool)
	accountRepo := repositories.NewAccountRepository(pool, queries)
	rentalRepo := repositories.NewRentalRepository(queries)

	farmer, plots, cropID := seedFarmerWithPlots(t, ctx, pool, 3)
	customer := seedCustomer(t, ctx, pool)

	for _, plot := range plots {
		if _, err := rentalRepo.CreateRental(ctx, plot, customer, cropID, 6); err != nil {
			t.Fatalf("renting plot: %v", err)
		}
	}

	recipients, err := accountRepo.GetCustomersOfFarmer(ctx, farmer)
	if err != nil {
		t.Fatalf("getting customers of farmer: %v", err)
	}

	if len(recipients) != 1 {
		t.Fatalf("got %d recipients for one customer renting 3 plots, want 1: %+v", len(recipients), recipients)
	}
	if recipients[0].AccountID != customer {
		t.Errorf("recipient = %v, want the renting customer %v", recipients[0].AccountID, customer)
	}
}

// TestGetCustomersOfFarmer_OnlyCurrentRenters checks the two exclusions the
// audience definition rests on: a customer whose rental has ended is no longer
// addressable, and another farmer's customers were never addressable.
func TestGetCustomersOfFarmer_OnlyCurrentRenters(t *testing.T) {
	pool := setupTestDB(t)
	ctx := context.Background()

	queries := database.New(pool)
	accountRepo := repositories.NewAccountRepository(pool, queries)
	rentalRepo := repositories.NewRentalRepository(queries)

	farmer, plots, cropID := seedFarmerWithPlots(t, ctx, pool, 2)
	otherFarmer, otherPlots, otherCrop := seedFarmerWithPlots(t, ctx, pool, 1)

	current := seedCustomer(t, ctx, pool)
	expired := seedCustomer(t, ctx, pool)
	somebodyElses := seedCustomer(t, ctx, pool)

	if _, err := rentalRepo.CreateRental(ctx, plots[0], current, cropID, 6); err != nil {
		t.Fatalf("renting plot: %v", err)
	}
	rentPast(t, ctx, pool, plots[1], expired, cropID)
	if _, err := rentalRepo.CreateRental(ctx, otherPlots[0], somebodyElses, otherCrop, 6); err != nil {
		t.Fatalf("renting other farmer's plot: %v", err)
	}

	recipients, err := accountRepo.GetCustomersOfFarmer(ctx, farmer)
	if err != nil {
		t.Fatalf("getting customers of farmer: %v", err)
	}

	if len(recipients) != 1 {
		t.Fatalf("got %d recipients, want only the current renter: %+v", len(recipients), recipients)
	}
	if recipients[0].AccountID != current {
		t.Errorf("recipient = %v, want the current renter %v", recipients[0].AccountID, current)
	}

	// And the other farmer reaches his own customer, not this farmer's.
	others, err := accountRepo.GetCustomersOfFarmer(ctx, otherFarmer)
	if err != nil {
		t.Fatalf("getting customers of the other farmer: %v", err)
	}
	if len(others) != 1 || others[0].AccountID != somebodyElses {
		t.Errorf("other farmer's recipients = %+v, want only %v", others, somebodyElses)
	}
}

// TestGetAnnouncementsForCustomer_OnlyFromFarmersCurrentlyRentedFrom checks
// the two properties the board actually has: it is scoped to the farmers the
// customer currently rents from, and each of their notices appears once
// however many plots he rents from them.
//
// It deliberately does not check that the board equals what he was mailed —
// it does not. The rental gates the farmer, not the notice, so notices posted
// before this customer's rental began are on his board too; see the comment on
// GetAnnouncementsForCustomer.
func TestGetAnnouncementsForCustomer_OnlyFromFarmersCurrentlyRentedFrom(t *testing.T) {
	pool := setupTestDB(t)
	ctx := context.Background()

	queries := database.New(pool)
	rentalRepo := repositories.NewRentalRepository(queries)
	announcementRepo := repositories.NewAnnouncementRepository(queries)

	farmer, plots, cropID := seedFarmerWithPlots(t, ctx, pool, 2)
	otherFarmer, _, _ := seedFarmerWithPlots(t, ctx, pool, 1)
	customer := seedCustomer(t, ctx, pool)

	// Renting two plots from the same farmer must not double his notices.
	for _, plot := range plots {
		if _, err := rentalRepo.CreateRental(ctx, plot, customer, cropID, 6); err != nil {
			t.Fatalf("renting plot: %v", err)
		}
	}

	if _, err := announcementRepo.CreateAnnouncement(ctx, farmer, "Ernte", "Samstag um 9"); err != nil {
		t.Fatalf("creating announcement: %v", err)
	}
	if _, err := announcementRepo.CreateAnnouncement(ctx, otherFarmer, "Fremd", "Nicht sichtbar"); err != nil {
		t.Fatalf("creating the other farmer's announcement: %v", err)
	}

	board, err := announcementRepo.GetAnnouncementsForCustomer(ctx, customer)
	if err != nil {
		t.Fatalf("getting the customer's board: %v", err)
	}

	if len(board) != 1 {
		t.Fatalf("board has %d notices, want 1: %+v", len(board), board)
	}
	if board[0].Subject != "Ernte" {
		t.Errorf("subject = %q, want the notice of the farmer rented from", board[0].Subject)
	}
	if board[0].FarmName != "Green Acres" {
		t.Errorf("farm name = %q, want it resolved for the reader", board[0].FarmName)
	}
}
