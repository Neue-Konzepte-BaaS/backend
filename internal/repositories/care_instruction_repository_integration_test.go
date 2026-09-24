package repositories_test

import (
	"context"
	"errors"
	"testing"

	"github.com/Neue-Konzepte-BaaS/backend/internal/repositories"
	database "github.com/Neue-Konzepte-BaaS/backend/internal/repositories/db"
	"github.com/Neue-Konzepte-BaaS/backend/internal/services"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// rentStartedDaysAgo writes a rental that began in the past and is still
// running, in the given status. The repository only ever *requests* a rental
// starting in the future, so a rental already some weeks in has to be written
// directly — which is the only way to observe a current_week other than 1.
// The status is a parameter because the care guide's whole audience rule is
// that only an approved rental carries one.
func rentStartedDaysAgo(t *testing.T, ctx context.Context, pool *pgxpool.Pool, plot, customer, crop uuid.UUID, daysAgo int, durationMonths int, status string) {
	t.Helper()

	_, err := pool.Exec(ctx, `
		INSERT INTO rental (plot, customer, crop, period, status, message, decided_at)
		VALUES ($1, $2, $3, tstzrange(
			CURRENT_TIMESTAMP - make_interval(days => $4::int),
			CURRENT_TIMESTAMP - make_interval(days => $4::int) + make_interval(months => $5::int)
		), $6, 'please', CURRENT_TIMESTAMP - make_interval(days => $4::int))`,
		plot, customer, crop, daysAgo, durationMonths, status)
	if err != nil {
		t.Fatalf("inserting running rental: %v", err)
	}
}

// TestCareInstructionRoundTrip covers the writes an admin makes: the week
// order a guide is read back in, an edit, and a delete that reports whether
// anything was actually removed.
func TestCareInstructionRoundTrip(t *testing.T) {
	pool := setupTestDB(t)
	ctx := context.Background()

	queries := database.New(pool)
	careRepo := repositories.NewCareInstructionRepository(queries)

	_, _, _, cropID := seedFarmerWithPlots(t, ctx, pool, 1)

	// Inserted out of order, so a guide that came back in insertion order
	// would fail here.
	if _, err := careRepo.CreateCareInstruction(ctx, cropID, 3, "Thin out", "Leave the strongest seedling."); err != nil {
		t.Fatalf("creating instruction: %v", err)
	}
	first, err := careRepo.CreateCareInstruction(ctx, cropID, 1, "Water in", "A full can per plot.")
	if err != nil {
		t.Fatalf("creating instruction: %v", err)
	}

	instructions, err := careRepo.GetCareInstructionsByCrop(ctx, cropID)
	if err != nil {
		t.Fatalf("getting instructions: %v", err)
	}
	if len(instructions) != 2 {
		t.Fatalf("got %d instructions, want 2", len(instructions))
	}
	if instructions[0].Week != 1 || instructions[1].Week != 3 {
		t.Errorf("weeks = %d, %d; want them in ascending order", instructions[0].Week, instructions[1].Week)
	}

	updated, err := careRepo.UpdateCareInstruction(ctx, first.ID, 2, "Water in well", "Two full cans per plot.")
	if err != nil {
		t.Fatalf("updating instruction: %v", err)
	}
	if updated.Week != 2 || updated.Title != "Water in well" {
		t.Errorf("updated instruction = week %d %q, want week 2 %q", updated.Week, updated.Title, "Water in well")
	}
	if updated.Crop != cropID {
		t.Errorf("update moved the instruction to crop %v, want it to stay on %v", updated.Crop, cropID)
	}

	if err := careRepo.DeleteCareInstruction(ctx, first.ID); err != nil {
		t.Fatalf("deleting instruction: %v", err)
	}
	if err := careRepo.DeleteCareInstruction(ctx, first.ID); !errors.Is(err, services.ErrNotFound) {
		t.Errorf("deleting it twice = %v, want ErrNotFound", err)
	}
}

// TestCareInstructionRejectsBadInput pins the two constraint violations the
// repository translates, rather than letting them surface as 500s.
func TestCareInstructionRejectsBadInput(t *testing.T) {
	pool := setupTestDB(t)
	ctx := context.Background()

	queries := database.New(pool)
	careRepo := repositories.NewCareInstructionRepository(queries)

	_, _, _, cropID := seedFarmerWithPlots(t, ctx, pool, 1)

	if _, err := careRepo.CreateCareInstruction(ctx, uuid.New(), 1, "Water in", "..."); !errors.Is(err, services.ErrNotFound) {
		t.Errorf("creating against an unknown crop = %v, want ErrNotFound", err)
	}
	if _, err := careRepo.CreateCareInstruction(ctx, cropID, 900, "Water in", "..."); !errors.Is(err, services.ErrInvalidCareInstruction) {
		t.Errorf("creating with week 900 = %v, want ErrInvalidCareInstruction", err)
	}
	if _, err := careRepo.UpdateCareInstruction(ctx, uuid.New(), 1, "Water in", "..."); !errors.Is(err, services.ErrNotFound) {
		t.Errorf("updating an unknown instruction = %v, want ErrNotFound", err)
	}
}

// TestGetCareInstructionsByCrops checks the read the tenant's care guide
// makes: several crops in one round trip, each keyed by its own crop.
func TestGetCareInstructionsByCrops(t *testing.T) {
	pool := setupTestDB(t)
	ctx := context.Background()

	queries := database.New(pool)
	careRepo := repositories.NewCareInstructionRepository(queries)

	_, _, _, tomatoes := seedFarmerWithPlots(t, ctx, pool, 1)
	_, _, _, beans := seedFarmerWithPlots(t, ctx, pool, 1)

	if _, err := careRepo.CreateCareInstruction(ctx, tomatoes, 1, "Water in", "..."); err != nil {
		t.Fatalf("creating instruction: %v", err)
	}
	if _, err := careRepo.CreateCareInstruction(ctx, tomatoes, 2, "Thin out", "..."); err != nil {
		t.Fatalf("creating instruction: %v", err)
	}
	if _, err := careRepo.CreateCareInstruction(ctx, beans, 1, "Set the canes", "..."); err != nil {
		t.Fatalf("creating instruction: %v", err)
	}

	byCrop, err := careRepo.GetCareInstructionsByCrops(ctx, []uuid.UUID{tomatoes, beans})
	if err != nil {
		t.Fatalf("getting instructions by crops: %v", err)
	}
	if len(byCrop[tomatoes]) != 2 {
		t.Errorf("tomatoes have %d instructions, want 2", len(byCrop[tomatoes]))
	}
	if len(byCrop[beans]) != 1 {
		t.Errorf("beans have %d instructions, want 1", len(byCrop[beans]))
	}
	if _, ok := byCrop[uuid.New()]; ok {
		t.Error("a crop nobody asked for turned up in the result")
	}
}

// TestCareInstructionsFollowTheirCrop covers the ON DELETE CASCADE: advice
// about a crop that has left the catalog has nothing left to be about. The
// crop is deletable precisely because a care instruction does not hold it
// back the way a rental does.
func TestCareInstructionsFollowTheirCrop(t *testing.T) {
	pool := setupTestDB(t)
	ctx := context.Background()

	queries := database.New(pool)
	careRepo := repositories.NewCareInstructionRepository(queries)
	cropRepo := repositories.NewCropRepository(pool, queries)

	crop, err := cropRepo.CreateCrop(ctx, "Radishes-"+uuid.NewString(), 2)
	if err != nil {
		t.Fatalf("creating crop: %v", err)
	}
	if _, err := careRepo.CreateCareInstruction(ctx, crop.ID, 1, "Sow thinly", "..."); err != nil {
		t.Fatalf("creating instruction: %v", err)
	}

	if err := cropRepo.DeleteCrop(ctx, crop.ID); err != nil {
		t.Fatalf("deleting crop: %v", err)
	}

	instructions, err := careRepo.GetCareInstructionsByCrop(ctx, crop.ID)
	if err != nil {
		t.Fatalf("getting instructions: %v", err)
	}
	if len(instructions) != 0 {
		t.Errorf("got %d instructions for a deleted crop, want 0", len(instructions))
	}
}

// TestGetActiveRentalsByCustomer is where the tenant's "week 3 of 13" comes
// from: only running rentals, and week numbers measured from the rental's own
// start rather than a calendar week.
func TestGetActiveRentalsByCustomer(t *testing.T) {
	pool := setupTestDB(t)
	ctx := context.Background()

	queries := database.New(pool)
	rentalRepo := repositories.NewRentalRepository(queries)

	_, _, plots, cropID := seedFarmerWithPlots(t, ctx, pool, 3)
	customer := seedCustomer(t, ctx, pool)

	// Three months, started 15 days ago: day 14 begins week 3.
	rentStartedDaysAgo(t, ctx, pool, plots[0], customer, cropID, 15, 3, "approved")
	// Started yesterday, so week 1.
	rentNow(t, ctx, pool, plots[1], customer, cropID, 3)
	// Already over, so not part of the care guide at all.
	rentPast(t, ctx, pool, plots[2], customer, cropID)

	rentals, err := rentalRepo.GetActiveRentalsByCustomer(ctx, customer)
	if err != nil {
		t.Fatalf("getting active rentals: %v", err)
	}
	if len(rentals) != 2 {
		t.Fatalf("got %d active rentals, want 2 (the expired one excluded)", len(rentals))
	}

	byPlot := map[uuid.UUID]int32{}
	for _, rental := range rentals {
		byPlot[rental.PlotID] = rental.CurrentWeek
		if rental.Crop.ID != cropID || rental.Crop.Name == "" {
			t.Errorf("rental carries crop %+v, want the rented crop with its name", rental.Crop)
		}
		if rental.PlotName == "" || rental.FieldName == "" {
			t.Errorf("rental carries plot %q on field %q, want both named", rental.PlotName, rental.FieldName)
		}
		// Three months is 13 weeks (92 days rounded up).
		if rental.TotalWeeks != 13 && rental.TotalWeeks != 14 {
			t.Errorf("total weeks = %d, want 13 or 14 for a three-month rental", rental.TotalWeeks)
		}
	}
	if byPlot[plots[0]] != 3 {
		t.Errorf("a rental 15 days in is in week %d, want 3", byPlot[plots[0]])
	}
	if byPlot[plots[1]] != 1 {
		t.Errorf("a rental starting today is in week %d, want 1", byPlot[plots[1]])
	}
}

// TestGetActiveRentalsByCustomer_OnlyApproved is the rule the care guide's
// audience rests on since rental requests: a row exists from the moment a
// customer asks for a plot, and a declined request keeps its row forever. A
// guide handed out on the period alone would reach both.
func TestGetActiveRentalsByCustomer_OnlyApproved(t *testing.T) {
	pool := setupTestDB(t)
	ctx := context.Background()

	queries := database.New(pool)
	rentalRepo := repositories.NewRentalRepository(queries)

	_, _, plots, cropID := seedFarmerWithPlots(t, ctx, pool, 3)
	customer := seedCustomer(t, ctx, pool)

	rentStartedDaysAgo(t, ctx, pool, plots[0], customer, cropID, 8, 3, "approved")
	rentStartedDaysAgo(t, ctx, pool, plots[1], customer, cropID, 8, 3, "requested")
	rentStartedDaysAgo(t, ctx, pool, plots[2], customer, cropID, 8, 3, "declined")

	rentals, err := rentalRepo.GetActiveRentalsByCustomer(ctx, customer)
	if err != nil {
		t.Fatalf("getting active rentals: %v", err)
	}
	if len(rentals) != 1 {
		t.Fatalf("got %d rentals, want only the approved one: %+v", len(rentals), rentals)
	}
	if rentals[0].PlotID != plots[0] {
		t.Errorf("rental is for plot %v, want the approved one %v", rentals[0].PlotID, plots[0])
	}
}
