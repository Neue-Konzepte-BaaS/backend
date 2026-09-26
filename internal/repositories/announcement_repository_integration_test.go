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

// seedFarmerWithPlots creates a farmer owning one farm and one field with
// plotCount plots, each offering one crop, and returns the farmer id, the
// farm id, the plots and that crop.
func seedFarmerWithPlots(t *testing.T, ctx context.Context, pool *pgxpool.Pool, plotCount int) (uuid.UUID, uuid.UUID, []uuid.UUID, uuid.UUID) {
	t.Helper()

	queries := database.New(pool)
	accountRepo := repositories.NewAccountRepository(pool, queries)
	farmRepo := repositories.NewFarmRepository(queries)
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

	return farmer.ID, farmID, plots, crop.ID
}

// rentPast books an approved, already-ended rental on plot. The repository
// only ever requests a rental starting in the future, so an expired one has
// to be written directly.
func rentPast(t *testing.T, ctx context.Context, pool *pgxpool.Pool, plot, customer, crop uuid.UUID) {
	t.Helper()

	_, err := pool.Exec(ctx, `
		INSERT INTO rental (plot, customer, crop, period, status, message, decided_at)
		VALUES ($1, $2, $3, tstzrange(CURRENT_TIMESTAMP - interval '2 months', CURRENT_TIMESTAMP - interval '1 month'), 'approved', 'please', CURRENT_TIMESTAMP - interval '2 months')`,
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

	farmer, _, plots, cropID := seedFarmerWithPlots(t, ctx, pool, 3)
	customer := seedCustomer(t, ctx, pool)

	for _, plot := range plots {
		rentNow(t, ctx, pool, plot, customer, cropID, 6)
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

	farmer, _, plots, cropID := seedFarmerWithPlots(t, ctx, pool, 2)
	otherFarmer, _, otherPlots, otherCrop := seedFarmerWithPlots(t, ctx, pool, 1)

	current := seedCustomer(t, ctx, pool)
	expired := seedCustomer(t, ctx, pool)
	somebodyElses := seedCustomer(t, ctx, pool)

	rentNow(t, ctx, pool, plots[0], current, cropID, 6)
	rentPast(t, ctx, pool, plots[1], expired, cropID)
	rentNow(t, ctx, pool, otherPlots[0], somebodyElses, otherCrop, 6)

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
	announcementRepo := repositories.NewAnnouncementRepository(queries)

	farmer, _, plots, cropID := seedFarmerWithPlots(t, ctx, pool, 2)
	otherFarmer, _, _, _ := seedFarmerWithPlots(t, ctx, pool, 1)
	customer := seedCustomer(t, ctx, pool)

	// Renting two plots from the same farmer must not double his notices.
	for _, plot := range plots {
		rentNow(t, ctx, pool, plot, customer, cropID, 6)
	}

	if _, err := announcementRepo.CreateAnnouncement(ctx, farmer, "Ernte", "Samstag um 9", nil, nil); err != nil {
		t.Fatalf("creating announcement: %v", err)
	}
	if _, err := announcementRepo.CreateAnnouncement(ctx, otherFarmer, "Fremd", "Nicht sichtbar", nil, nil); err != nil {
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

// addField creates one more field with one plot on an existing farm, and
// returns their ids. Used to prove a scoped announcement/audience does not
// leak across fields belonging to the same farmer.
func addField(t *testing.T, ctx context.Context, pool *pgxpool.Pool, farmID uuid.UUID) (uuid.UUID, uuid.UUID) {
	t.Helper()

	queries := database.New(pool)
	fieldRepo := repositories.NewFieldRepository(queries)
	plotRepo := repositories.NewPlotRepository(queries)

	fieldID, err := fieldRepo.CreateField(ctx, models.Field{
		Name:        "Field 2",
		Farm:        farmID,
		Coordinates: rectangle(200, 200, 300, 300),
	})
	if err != nil {
		t.Fatalf("creating second field: %v", err)
	}

	plot, err := plotRepo.CreatePlot(ctx, models.Plot{
		Name:        "Plot",
		Field:       fieldID,
		Coordinates: rectangle(201, 201, 205, 205),
	})
	if err != nil {
		t.Fatalf("creating plot on second field: %v", err)
	}

	return fieldID, plot.ID
}

// TestGetAnnouncementsForCustomer_ScopedToFieldOnlyReachesThatFieldsRenters
// checks the property the whole scoping feature rests on: a renter of a
// different field of the same farm must not see a notice scoped to a field
// he does not rent on, even though he is a current customer of the farmer.
func TestGetAnnouncementsForCustomer_ScopedToFieldOnlyReachesThatFieldsRenters(t *testing.T) {
	pool := setupTestDB(t)
	ctx := context.Background()

	queries := database.New(pool)
	announcementRepo := repositories.NewAnnouncementRepository(queries)
	plotRepo := repositories.NewPlotRepository(queries)

	farmer, farmID, plots, cropID := seedFarmerWithPlots(t, ctx, pool, 1)
	_, otherPlot := addField(t, ctx, pool, farmID)

	inField := seedCustomer(t, ctx, pool)
	outsideField := seedCustomer(t, ctx, pool)

	rentNow(t, ctx, pool, plots[0], inField, cropID, 6)
	rentNow(t, ctx, pool, otherPlot, outsideField, cropID, 6)

	// seedFarmerWithPlots does not return its field id directly, so resolve
	// the field to scope the announcement to via the plot it seeded.
	targetField, err := plotRepo.GetPlotField(ctx, plots[0])
	if err != nil {
		t.Fatalf("looking up scoped field: %v", err)
	}

	if _, err := announcementRepo.CreateAnnouncement(ctx, farmer, "Nur Feld 1", "Sichtbar nur hier", &targetField, nil); err != nil {
		t.Fatalf("creating scoped announcement: %v", err)
	}

	board, err := announcementRepo.GetAnnouncementsForCustomer(ctx, inField)
	if err != nil {
		t.Fatalf("getting in-field customer's board: %v", err)
	}
	if len(board) != 1 {
		t.Fatalf("in-field renter's board has %d notices, want 1: %+v", len(board), board)
	}

	otherBoard, err := announcementRepo.GetAnnouncementsForCustomer(ctx, outsideField)
	if err != nil {
		t.Fatalf("getting outside-field customer's board: %v", err)
	}
	if len(otherBoard) != 0 {
		t.Fatalf("outside-field renter's board has %d notices, want 0 (scope must exclude him): %+v", len(otherBoard), otherBoard)
	}
}

// TestGetCustomersOfFarmerForField_DedupsAndExcludesOtherFields is the
// GetCustomersOfFarmer DISTINCT test's counterpart for the scoped audience
// query: a customer renting two plots of the scoped field is mailed once,
// and a customer of a different field of the same farm is not mailed at all.
func TestGetCustomersOfFarmerForField_DedupsAndExcludesOtherFields(t *testing.T) {
	pool := setupTestDB(t)
	ctx := context.Background()

	queries := database.New(pool)
	rentalRepo := repositories.NewRentalRepository(queries)
	accountRepo := repositories.NewAccountRepository(pool, queries)
	plotRepo := repositories.NewPlotRepository(queries)

	_, farmID, plots, cropID := seedFarmerWithPlots(t, ctx, pool, 2)
	_, otherPlot := addField(t, ctx, pool, farmID)

	twicePlotted := seedCustomer(t, ctx, pool)
	elsewhere := seedCustomer(t, ctx, pool)

	for _, plot := range plots {
		rentApprovedNow(t, ctx, rentalRepo, plot, twicePlotted, cropID)
	}
	rentApprovedNow(t, ctx, rentalRepo, otherPlot, elsewhere, cropID)

	targetField, err := plotRepo.GetPlotField(ctx, plots[0])
	if err != nil {
		t.Fatalf("looking up field: %v", err)
	}

	recipients, err := accountRepo.GetCustomersOfFarmerForField(ctx, targetField)
	if err != nil {
		t.Fatalf("getting customers of field: %v", err)
	}
	if len(recipients) != 1 {
		t.Fatalf("got %d recipients for one customer renting 2 plots of the field, want 1: %+v", len(recipients), recipients)
	}
	if recipients[0].AccountID != twicePlotted {
		t.Errorf("recipient = %v, want %v", recipients[0].AccountID, twicePlotted)
	}
}
