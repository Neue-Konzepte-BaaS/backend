package repositories_test

import (
	"context"
	"testing"

	"github.com/Neue-Konzepte-BaaS/backend/internal/repositories"
	database "github.com/Neue-Konzepte-BaaS/backend/internal/repositories/db"
	"github.com/google/uuid"
)

// TestCreateRipenessNotice_ResolvesFarmPlotAndCropNames checks that the
// insert query's joins actually resolve the names the response and inbox
// need, not just the ids.
func TestCreateRipenessNotice_ResolvesFarmPlotAndCropNames(t *testing.T) {
	pool := setupTestDB(t)
	ctx := context.Background()

	queries := database.New(pool)
	ripenessRepo := repositories.NewRipenessNoticeRepository(queries)

	farmer, _, plots, cropID := seedFarmerWithPlots(t, ctx, pool, 1)

	notice, err := ripenessRepo.CreateRipenessNotice(ctx, farmer, plots[0], cropID)
	if err != nil {
		t.Fatalf("creating ripeness notice: %v", err)
	}

	if notice.FarmName != "Green Acres" {
		t.Errorf("farm name = %q, want %q", notice.FarmName, "Green Acres")
	}
	if notice.PlotName == "" {
		t.Error("plot name is empty, want it resolved")
	}
	if notice.CropName == "" {
		t.Error("crop name is empty, want it resolved")
	}
}

// TestGetCustomersOfFarmerForPlotAndCrop_MatchesPlotAndCropOnly is the
// audience test for the ripeness notice query: a customer renting the
// target plot with the target crop is mailed, a customer on the same plot
// growing a different crop is excluded, and a customer on a different plot
// growing the target crop is excluded.
func TestGetCustomersOfFarmerForPlotAndCrop_MatchesPlotAndCropOnly(t *testing.T) {
	pool := setupTestDB(t)
	ctx := context.Background()

	queries := database.New(pool)
	rentalRepo := repositories.NewRentalRepository(queries)
	accountRepo := repositories.NewAccountRepository(pool, queries)
	cropRepo := repositories.NewCropRepository(pool, queries)

	_, _, plots, cropID := seedFarmerWithPlots(t, ctx, pool, 3)

	otherCropName := "Kohl-" + uuid.NewString()
	otherCrop, err := cropRepo.CreateCrop(ctx, otherCropName, otherCropName, 4)
	if err != nil {
		t.Fatalf("creating other crop: %v", err)
	}

	matching := seedCustomer(t, ctx, pool)
	wrongCrop := seedCustomer(t, ctx, pool)
	wrongPlot := seedCustomer(t, ctx, pool)

	// Target plot, target crop: must be reached.
	rentApprovedNow(t, ctx, rentalRepo, plots[0], matching, cropID)
	// Target plot's sibling, different crop: must be excluded.
	rentApprovedNow(t, ctx, rentalRepo, plots[1], wrongCrop, otherCrop.ID)
	// A third plot, target crop: must be excluded — same crop, wrong plot.
	rentApprovedNow(t, ctx, rentalRepo, plots[2], wrongPlot, cropID)

	recipients, err := accountRepo.GetCustomersOfFarmerForPlotAndCrop(ctx, plots[0], cropID)
	if err != nil {
		t.Fatalf("getting customers for plot and crop: %v", err)
	}
	if len(recipients) != 1 {
		t.Fatalf("got %d recipients, want 1 (wrong crop and wrong plot excluded): %+v", len(recipients), recipients)
	}
	if recipients[0].AccountID != matching {
		t.Errorf("recipient = %v, want %v", recipients[0].AccountID, matching)
	}
}

// TestGetRipenessNoticesForCustomer_OnlyMatchingPlotAndCrop checks the
// customer-facing read side: a notice only shows up for a customer whose
// active rental covers the notice's plot with the notice's crop.
func TestGetRipenessNoticesForCustomer_OnlyMatchingPlotAndCrop(t *testing.T) {
	pool := setupTestDB(t)
	ctx := context.Background()

	queries := database.New(pool)
	rentalRepo := repositories.NewRentalRepository(queries)
	ripenessRepo := repositories.NewRipenessNoticeRepository(queries)

	farmer, _, plots, cropID := seedFarmerWithPlots(t, ctx, pool, 2)

	matching := seedCustomer(t, ctx, pool)
	wrongPlot := seedCustomer(t, ctx, pool)

	rentApprovedNow(t, ctx, rentalRepo, plots[0], matching, cropID)
	rentApprovedNow(t, ctx, rentalRepo, plots[1], wrongPlot, cropID)

	if _, err := ripenessRepo.CreateRipenessNotice(ctx, farmer, plots[0], cropID); err != nil {
		t.Fatalf("creating ripeness notice: %v", err)
	}

	matchingBoard, err := ripenessRepo.GetRipenessNoticesForCustomer(ctx, matching)
	if err != nil {
		t.Fatalf("getting matching customer's notices: %v", err)
	}
	if len(matchingBoard) != 1 {
		t.Fatalf("matching customer's notices = %d, want 1: %+v", len(matchingBoard), matchingBoard)
	}

	wrongPlotBoard, err := ripenessRepo.GetRipenessNoticesForCustomer(ctx, wrongPlot)
	if err != nil {
		t.Fatalf("getting wrong-plot customer's notices: %v", err)
	}
	if len(wrongPlotBoard) != 0 {
		t.Fatalf("wrong-plot customer's notices = %d, want 0: %+v", len(wrongPlotBoard), wrongPlotBoard)
	}
}
