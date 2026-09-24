package services

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/Neue-Konzepte-BaaS/backend/internal/models"
	"github.com/google/uuid"
)

// fakeCareInstructionRepo serves a fixed guide per crop and records what the
// service asked for.
type fakeCareInstructionRepo struct {
	byCrop    map[uuid.UUID][]models.CareInstruction
	err       error
	gotCrops  []uuid.UUID
	cropCalls int
}

func (f *fakeCareInstructionRepo) CreateCareInstruction(_ context.Context, crop uuid.UUID, week int32, title, body string) (models.CareInstruction, error) {
	if f.err != nil {
		return models.CareInstruction{}, f.err
	}
	return models.CareInstruction{ID: uuid.New(), Crop: crop, Week: week, Title: title, Body: body}, nil
}

func (f *fakeCareInstructionRepo) UpdateCareInstruction(_ context.Context, id uuid.UUID, week int32, title, body string) (models.CareInstruction, error) {
	if f.err != nil {
		return models.CareInstruction{}, f.err
	}
	return models.CareInstruction{ID: id, Week: week, Title: title, Body: body}, nil
}

func (f *fakeCareInstructionRepo) DeleteCareInstruction(context.Context, uuid.UUID) error {
	return f.err
}

func (f *fakeCareInstructionRepo) GetCareInstructionsByCrop(_ context.Context, crop uuid.UUID) ([]models.CareInstruction, error) {
	if f.err != nil {
		return nil, f.err
	}
	return f.byCrop[crop], nil
}

func (f *fakeCareInstructionRepo) GetCareInstructionsByCrops(_ context.Context, crops []uuid.UUID) (map[uuid.UUID][]models.CareInstruction, error) {
	f.cropCalls++
	f.gotCrops = crops
	if f.err != nil {
		return nil, f.err
	}
	out := make(map[uuid.UUID][]models.CareInstruction, len(crops))
	for _, crop := range crops {
		if instructions, ok := f.byCrop[crop]; ok {
			out[crop] = instructions
		}
	}
	return out, nil
}

// fakeCareRentalRepo only has to answer the one read the care guide makes; the
// embedded interface keeps the other methods off the test's back.
type fakeCareRentalRepo struct {
	RentalRepository

	active []models.ActiveRental
	err    error
}

func (f *fakeCareRentalRepo) GetActiveRentalsByCustomer(context.Context, uuid.UUID) ([]models.ActiveRental, error) {
	if f.err != nil {
		return nil, f.err
	}
	return f.active, nil
}

func activeRental(plotName string, crop models.Crop, currentWeek, totalWeeks int32) models.ActiveRental {
	return models.ActiveRental{
		Rental: models.Rental{
			ID:      uuid.New(),
			PlotID:  uuid.New(),
			CropID:  crop.ID,
			StartAt: time.Now().AddDate(0, 0, -7*int(currentWeek-1)),
			EndAt:   time.Now().AddDate(0, 0, 7*int(totalWeeks-currentWeek+1)),
		},
		PlotName:    plotName,
		FieldName:   "North field",
		Crop:        crop,
		CurrentWeek: currentWeek,
		TotalWeeks:  totalWeeks,
	}
}

func instruction(crop uuid.UUID, week int32, title string) models.CareInstruction {
	return models.CareInstruction{ID: uuid.New(), Crop: crop, Week: week, Title: title, Body: "..."}
}

func TestGetCareGuideForCustomer(t *testing.T) {
	tomatoes := models.Crop{ID: uuid.New(), Name: "Tomatoes", DurationMonths: 3}
	beans := models.Crop{ID: uuid.New(), Name: "Beans", DurationMonths: 3}

	t.Run("pairs each rented plot with its crop's guide", func(t *testing.T) {
		careRepo := &fakeCareInstructionRepo{byCrop: map[uuid.UUID][]models.CareInstruction{
			tomatoes.ID: {instruction(tomatoes.ID, 1, "Water in"), instruction(tomatoes.ID, 2, "Thin out")},
			beans.ID:    {instruction(beans.ID, 1, "Set the canes")},
		}}
		rentalRepo := &fakeCareRentalRepo{active: []models.ActiveRental{
			activeRental("Plot 1", tomatoes, 2, 13),
			activeRental("Plot 2", beans, 2, 13),
		}}
		service := NewCareGuideService(careRepo, rentalRepo)

		guides, err := service.GetCareGuideForCustomer(context.Background(), uuid.New())
		if err != nil {
			t.Fatalf("GetCareGuideForCustomer: %v", err)
		}
		if len(guides) != 2 {
			t.Fatalf("got %d guides, want 2", len(guides))
		}
		if guides[0].PlotName != "Plot 1" || len(guides[0].Instructions) != 2 {
			t.Errorf("first guide = %q with %d instructions, want Plot 1 with 2", guides[0].PlotName, len(guides[0].Instructions))
		}
		if guides[1].Crop.Name != "Beans" || len(guides[1].Instructions) != 1 {
			t.Errorf("second guide = %q with %d instructions, want Beans with 1", guides[1].Crop.Name, len(guides[1].Instructions))
		}
		if guides[0].CurrentWeek != 2 || guides[0].TotalWeeks != 13 {
			t.Errorf("week numbers = %d/%d, want 2/13", guides[0].CurrentWeek, guides[0].TotalWeeks)
		}
	})

	t.Run("reads each crop's guide once however many plots grow it", func(t *testing.T) {
		careRepo := &fakeCareInstructionRepo{byCrop: map[uuid.UUID][]models.CareInstruction{
			tomatoes.ID: {instruction(tomatoes.ID, 1, "Water in")},
		}}
		rentalRepo := &fakeCareRentalRepo{active: []models.ActiveRental{
			activeRental("Plot 1", tomatoes, 1, 13),
			activeRental("Plot 2", tomatoes, 1, 13),
			activeRental("Plot 3", tomatoes, 1, 13),
		}}
		service := NewCareGuideService(careRepo, rentalRepo)

		guides, err := service.GetCareGuideForCustomer(context.Background(), uuid.New())
		if err != nil {
			t.Fatalf("GetCareGuideForCustomer: %v", err)
		}
		if careRepo.cropCalls != 1 {
			t.Errorf("instruction repo called %d times, want 1", careRepo.cropCalls)
		}
		if len(careRepo.gotCrops) != 1 {
			t.Errorf("asked for %d crops, want 1 (deduplicated)", len(careRepo.gotCrops))
		}
		for i, guide := range guides {
			if len(guide.Instructions) != 1 {
				t.Errorf("guide %d has %d instructions, want 1", i, len(guide.Instructions))
			}
		}
	})

	t.Run("drops weeks the rental never reaches", func(t *testing.T) {
		careRepo := &fakeCareInstructionRepo{byCrop: map[uuid.UUID][]models.CareInstruction{
			tomatoes.ID: {
				instruction(tomatoes.ID, 1, "Water in"),
				instruction(tomatoes.ID, 13, "Harvest"),
				instruction(tomatoes.ID, 30, "Never happens"),
			},
		}}
		rentalRepo := &fakeCareRentalRepo{active: []models.ActiveRental{activeRental("Plot 1", tomatoes, 1, 13)}}
		service := NewCareGuideService(careRepo, rentalRepo)

		guides, err := service.GetCareGuideForCustomer(context.Background(), uuid.New())
		if err != nil {
			t.Fatalf("GetCareGuideForCustomer: %v", err)
		}
		if len(guides[0].Instructions) != 2 {
			t.Fatalf("got %d instructions, want 2", len(guides[0].Instructions))
		}
		for _, got := range guides[0].Instructions {
			if got.Week > 13 {
				t.Errorf("instruction for week %d survived a 13-week rental", got.Week)
			}
		}
	})

	t.Run("returns a plot whose crop has no guide yet", func(t *testing.T) {
		careRepo := &fakeCareInstructionRepo{byCrop: map[uuid.UUID][]models.CareInstruction{}}
		rentalRepo := &fakeCareRentalRepo{active: []models.ActiveRental{activeRental("Plot 1", tomatoes, 1, 13)}}
		service := NewCareGuideService(careRepo, rentalRepo)

		guides, err := service.GetCareGuideForCustomer(context.Background(), uuid.New())
		if err != nil {
			t.Fatalf("GetCareGuideForCustomer: %v", err)
		}
		if len(guides) != 1 {
			t.Fatalf("got %d guides, want 1", len(guides))
		}
		if guides[0].Instructions == nil {
			t.Error("instructions are nil; want an empty slice so the response encodes []")
		}
	})

	t.Run("renting nothing asks for no guides at all", func(t *testing.T) {
		careRepo := &fakeCareInstructionRepo{}
		service := NewCareGuideService(careRepo, &fakeCareRentalRepo{})

		guides, err := service.GetCareGuideForCustomer(context.Background(), uuid.New())
		if err != nil {
			t.Fatalf("GetCareGuideForCustomer: %v", err)
		}
		if len(guides) != 0 {
			t.Errorf("got %d guides, want 0", len(guides))
		}
		if careRepo.cropCalls != 0 {
			t.Errorf("instruction repo called %d times, want 0", careRepo.cropCalls)
		}
	})

	t.Run("reports a failing rental read", func(t *testing.T) {
		service := NewCareGuideService(&fakeCareInstructionRepo{}, &fakeCareRentalRepo{err: errors.New("boom")})

		if _, err := service.GetCareGuideForCustomer(context.Background(), uuid.New()); err == nil {
			t.Fatal("GetCareGuideForCustomer succeeded, want an error")
		}
	})
}

func TestCareInstructionWritesPassSentinelsThrough(t *testing.T) {
	ctx := context.Background()

	t.Run("create reports an unknown crop as not found", func(t *testing.T) {
		service := NewCareGuideService(&fakeCareInstructionRepo{err: ErrNotFound}, &fakeCareRentalRepo{})

		_, err := service.CreateCareInstruction(ctx, uuid.New(), 1, "Water in", "...")
		if !errors.Is(err, ErrNotFound) {
			t.Errorf("err = %v, want ErrNotFound", err)
		}
	})

	t.Run("update reports an unknown instruction as not found", func(t *testing.T) {
		service := NewCareGuideService(&fakeCareInstructionRepo{err: ErrNotFound}, &fakeCareRentalRepo{})

		_, err := service.UpdateCareInstruction(ctx, uuid.New(), 1, "Water in", "...")
		if !errors.Is(err, ErrNotFound) {
			t.Errorf("err = %v, want ErrNotFound", err)
		}
	})

	t.Run("delete reports an unknown instruction as not found", func(t *testing.T) {
		service := NewCareGuideService(&fakeCareInstructionRepo{err: ErrNotFound}, &fakeCareRentalRepo{})

		if err := service.DeleteCareInstruction(ctx, uuid.New()); !errors.Is(err, ErrNotFound) {
			t.Errorf("err = %v, want ErrNotFound", err)
		}
	})

	t.Run("a rejected week stays ErrInvalidCareInstruction", func(t *testing.T) {
		service := NewCareGuideService(&fakeCareInstructionRepo{err: ErrInvalidCareInstruction}, &fakeCareRentalRepo{})

		_, err := service.CreateCareInstruction(ctx, uuid.New(), 900, "Water in", "...")
		if !errors.Is(err, ErrInvalidCareInstruction) {
			t.Errorf("err = %v, want ErrInvalidCareInstruction", err)
		}
	})

	t.Run("an unexpected repository error is wrapped, not classified", func(t *testing.T) {
		service := NewCareGuideService(&fakeCareInstructionRepo{err: errors.New("boom")}, &fakeCareRentalRepo{})

		_, err := service.CreateCareInstruction(ctx, uuid.New(), 1, "Water in", "...")
		if err == nil || errors.Is(err, ErrNotFound) {
			t.Errorf("err = %v, want a plain wrapped error", err)
		}
	})
}
