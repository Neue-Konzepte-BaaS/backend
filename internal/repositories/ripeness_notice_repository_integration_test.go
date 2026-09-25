package repositories_test

import (
	"context"
	"testing"

	"github.com/Neue-Konzepte-BaaS/backend/internal/repositories"
	database "github.com/Neue-Konzepte-BaaS/backend/internal/repositories/db"
	"github.com/google/uuid"
)

// TestCreateRipenessNotice_ResolvesFarmFieldAndCropNames checks that the
// insert query's joins actually resolve the names the response and inbox
// need, not just the ids.
func TestCreateRipenessNotice_ResolvesFarmFieldAndCropNames(t *testing.T) {
	pool := setupTestDB(t)
	ctx := context.Background()

	queries := database.New(pool)
	plotRepo := repositories.NewPlotRepository(queries)
	ripenessRepo := repositories.NewRipenessNoticeRepository(queries)

	farmer, _, plots, cropID := seedFarmerWithPlots(t, ctx, pool, 1)
	fieldID, err := plotRepo.GetPlotField(ctx, plots[0])
	if err != nil {
		t.Fatalf("looking up field: %v", err)
	}

	notice, err := ripenessRepo.CreateRipenessNotice(ctx, farmer, fieldID, cropID)
	if err != nil {
		t.Fatalf("creating ripeness notice: %v", err)
	}

	if notice.FarmName != "Green Acres" {
		t.Errorf("farm name = %q, want %q", notice.FarmName, "Green Acres")
	}
	if notice.FieldName != "Field 1" {
		t.Errorf("field name = %q, want %q", notice.FieldName, "Field 1")
	}
	if notice.CropName == "" {
		t.Error("crop name is empty, want it resolved")
	}
}

// TestGetCustomersOfFarmerForFieldAndCrop_DedupsMatchesFieldAndCropOnly is the
// DISTINCT test for the ripeness audience query: a customer renting two
// matching plots is mailed once, a customer growing a different crop on the
// same field is excluded, and a customer on a different field is excluded.
func TestGetCustomersOfFarmerForFieldAndCrop_DedupsMatchesFieldAndCropOnly(t *testing.T) {
	pool := setupTestDB(t)
	ctx := context.Background()

	queries := database.New(pool)
	rentalRepo := repositories.NewRentalRepository(queries)
	accountRepo := repositories.NewAccountRepository(pool, queries)
	plotRepo := repositories.NewPlotRepository(queries)
	cropRepo := repositories.NewCropRepository(pool, queries)

	_, farmID, plots, cropID := seedFarmerWithPlots(t, ctx, pool, 3)
	_, otherFieldPlot := addField(t, ctx, pool, farmID)

	otherCrop, err := cropRepo.CreateCrop(ctx, "Kohl-"+uuid.NewString(), 4)
	if err != nil {
		t.Fatalf("creating other crop: %v", err)
	}

	matchingTwice := seedCustomer(t, ctx, pool)
	wrongCrop := seedCustomer(t, ctx, pool)
	wrongField := seedCustomer(t, ctx, pool)

	// Two plots of the target field, growing the target crop: must count once.
	for _, plot := range plots[:2] {
		rentApprovedNow(t, ctx, rentalRepo, plot, matchingTwice, cropID)
	}
	// Same field, different crop: must be excluded.
	rentApprovedNow(t, ctx, rentalRepo, plots[2], wrongCrop, otherCrop.ID)
	// Different field, same crop: must be excluded.
	rentApprovedNow(t, ctx, rentalRepo, otherFieldPlot, wrongField, cropID)

	targetField, err := plotRepo.GetPlotField(ctx, plots[0])
	if err != nil {
		t.Fatalf("looking up field: %v", err)
	}

	recipients, err := accountRepo.GetCustomersOfFarmerForFieldAndCrop(ctx, targetField, cropID)
	if err != nil {
		t.Fatalf("getting customers for field and crop: %v", err)
	}
	if len(recipients) != 1 {
		t.Fatalf("got %d recipients, want 1 (deduped, wrong crop and wrong field excluded): %+v", len(recipients), recipients)
	}
	if recipients[0].AccountID != matchingTwice {
		t.Errorf("recipient = %v, want %v", recipients[0].AccountID, matchingTwice)
	}
}

// TestGetRipenessNoticesForCustomer_OnlyMatchingFieldAndCrop checks the
// customer-facing read side: a notice only shows up for a customer whose
// active rental covers the notice's field with the notice's crop.
func TestGetRipenessNoticesForCustomer_OnlyMatchingFieldAndCrop(t *testing.T) {
	pool := setupTestDB(t)
	ctx := context.Background()

	queries := database.New(pool)
	rentalRepo := repositories.NewRentalRepository(queries)
	plotRepo := repositories.NewPlotRepository(queries)
	ripenessRepo := repositories.NewRipenessNoticeRepository(queries)

	farmer, farmID, plots, cropID := seedFarmerWithPlots(t, ctx, pool, 1)
	_, otherFieldPlot := addField(t, ctx, pool, farmID)

	matching := seedCustomer(t, ctx, pool)
	wrongField := seedCustomer(t, ctx, pool)

	rentApprovedNow(t, ctx, rentalRepo, plots[0], matching, cropID)
	rentApprovedNow(t, ctx, rentalRepo, otherFieldPlot, wrongField, cropID)

	fieldID, err := plotRepo.GetPlotField(ctx, plots[0])
	if err != nil {
		t.Fatalf("looking up field: %v", err)
	}
	if _, err := ripenessRepo.CreateRipenessNotice(ctx, farmer, fieldID, cropID); err != nil {
		t.Fatalf("creating ripeness notice: %v", err)
	}

	matchingBoard, err := ripenessRepo.GetRipenessNoticesForCustomer(ctx, matching)
	if err != nil {
		t.Fatalf("getting matching customer's notices: %v", err)
	}
	if len(matchingBoard) != 1 {
		t.Fatalf("matching customer's notices = %d, want 1: %+v", len(matchingBoard), matchingBoard)
	}

	wrongFieldBoard, err := ripenessRepo.GetRipenessNoticesForCustomer(ctx, wrongField)
	if err != nil {
		t.Fatalf("getting wrong-field customer's notices: %v", err)
	}
	if len(wrongFieldBoard) != 0 {
		t.Fatalf("wrong-field customer's notices = %d, want 0: %+v", len(wrongFieldBoard), wrongFieldBoard)
	}
}
