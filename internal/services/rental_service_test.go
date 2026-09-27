package services

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/Neue-Konzepte-BaaS/backend/internal/models"
	"github.com/google/uuid"
)

// fakeRentalRepo is an in-memory RentalRepository for exercising the service
// layer without a database.
type fakeRentalRepo struct {
	createErr  error
	rentalByID map[uuid.UUID]models.RentalWithField
	getErr     error
	updateErr  error
	updated    models.Rental
}

func (f *fakeRentalRepo) CreateRentalRequest(_ context.Context, plot, customer, crop uuid.UUID, startAt time.Time, durationMonths int32, message string) (models.Rental, error) {
	if f.createErr != nil {
		return models.Rental{}, f.createErr
	}
	return models.Rental{
		ID:       uuid.New(),
		PlotID:   plot,
		CropID:   crop,
		Customer: customer,
		StartAt:  startAt,
		EndAt:    startAt.AddDate(0, int(durationMonths), 0),
		Status:   models.RentalStatusRequested,
		Message:  message,
	}, nil
}

func (f *fakeRentalRepo) UpdateRentalStatus(_ context.Context, id uuid.UUID, status models.RentalStatus) (models.Rental, error) {
	if f.updateErr != nil {
		return models.Rental{}, f.updateErr
	}
	rental := f.updated
	rental.ID = id
	rental.Status = status
	return rental, nil
}

func (f *fakeRentalRepo) GetRentalWithFieldByID(_ context.Context, id uuid.UUID) (models.RentalWithField, error) {
	if f.getErr != nil {
		return models.RentalWithField{}, f.getErr
	}
	if rental, ok := f.rentalByID[id]; ok {
		return rental, nil
	}
	return models.RentalWithField{}, ErrNotFound
}

func (f *fakeRentalRepo) GetRentalsByCustomer(context.Context, uuid.UUID) ([]models.RentalWithPlot, error) {
	return nil, nil
}

func (f *fakeRentalRepo) GetRentalsByFarm(context.Context, uuid.UUID) ([]models.RentalWithPlotAndCustomer, error) {
	return nil, nil
}

func (f *fakeRentalRepo) GetRentalByID(context.Context, uuid.UUID) (models.Rental, error) {
	return models.Rental{}, ErrNotFound
}

func (f *fakeRentalRepo) IsPlotAvailable(context.Context, uuid.UUID, time.Time, int32) (bool, error) {
	return true, nil
}

// Unused by rentalService — the care guide is what reads this; see
// fakeCareRentalRepo in care_guide_service_test.go.
func (f *fakeRentalRepo) GetActiveRentalsByCustomer(context.Context, uuid.UUID) ([]models.ActiveRental, error) {
	return nil, nil
}

// fakePlotRepo and fakeFieldRepo now live in announcement_service_test.go
// (shared across this package's tests) -- see that file for GetPlotByID and
// the farm-filter-aware GetNearestPlots this service's tests also rely on.

// fakeCropRepo is an in-memory CropRepository, only implementing what
// rentalService needs.
type fakeCropRepo struct {
	crop           models.Crop
	cropErr        error
	offeredCropIDs []uuid.UUID
}

func (f *fakeCropRepo) CreateCrop(context.Context, string, int32) (models.Crop, error) {
	return models.Crop{}, nil
}
func (f *fakeCropRepo) UpdateCrop(context.Context, uuid.UUID, string, int32) (models.Crop, error) {
	return models.Crop{}, nil
}
func (f *fakeCropRepo) DeleteCrop(context.Context, uuid.UUID) error { return nil }
func (f *fakeCropRepo) GetAllCrops(context.Context) ([]models.Crop, error) {
	return nil, nil
}
func (f *fakeCropRepo) GetCropByID(_ context.Context, _ uuid.UUID) (models.Crop, error) {
	if f.cropErr != nil {
		return models.Crop{}, f.cropErr
	}
	return f.crop, nil
}
func (f *fakeCropRepo) SetPlotCrops(context.Context, uuid.UUID, int32, []uuid.UUID) error {
	return nil
}
func (f *fakeCropRepo) GetCropsByPlot(_ context.Context, _ uuid.UUID) ([]models.Crop, error) {
	crops := make([]models.Crop, len(f.offeredCropIDs))
	for i, id := range f.offeredCropIDs {
		crops[i] = models.Crop{ID: id}
	}
	return crops, nil
}
func (f *fakeCropRepo) GetCropsByPlots(context.Context, []uuid.UUID) (map[uuid.UUID][]models.Crop, error) {
	return nil, nil
}
func (f *fakeCropRepo) GetPricedCropOfferingsByPlots(context.Context, []uuid.UUID) (map[uuid.UUID][]models.PlotCropOffering, error) {
	return nil, nil
}

// fakeSeasonRepo is an in-memory SeasonRepository, only implementing what
// rentalService needs: resolving the effective season for a (crop, farm)
// pair.
type fakeSeasonRepo struct {
	// seasonByCropAndFarm holds the rule a farm sees for a crop, keyed by
	// crop and then farm. A crop absent here, or a farm absent under it, is
	// unrestricted for that farm.
	seasonByCropAndFarm map[uuid.UUID]map[uuid.UUID]models.Season
}

func (f *fakeSeasonRepo) CreateSeason(context.Context, *uuid.UUID, string, int32, int32, int32, int32) (models.Season, error) {
	return models.Season{}, nil
}
func (f *fakeSeasonRepo) UpdateSeason(context.Context, uuid.UUID, string, int32, int32, int32, int32) (models.Season, error) {
	return models.Season{}, nil
}
func (f *fakeSeasonRepo) DeleteSeason(context.Context, uuid.UUID) error { return nil }
func (f *fakeSeasonRepo) GetSeasonByID(context.Context, uuid.UUID) (models.Season, error) {
	return models.Season{}, nil
}
func (f *fakeSeasonRepo) GetDefaultSeasons(context.Context) ([]models.Season, error) { return nil, nil }
func (f *fakeSeasonRepo) GetFarmSeasons(context.Context, uuid.UUID) ([]models.Season, error) {
	return nil, nil
}
func (f *fakeSeasonRepo) CreateCropSeasonRule(context.Context, uuid.UUID, *uuid.UUID, uuid.UUID) (models.CropSeasonRule, error) {
	return models.CropSeasonRule{}, nil
}
func (f *fakeSeasonRepo) UpdateCropSeasonRule(context.Context, uuid.UUID, uuid.UUID) (models.CropSeasonRule, error) {
	return models.CropSeasonRule{}, nil
}
func (f *fakeSeasonRepo) DeleteCropSeasonRule(context.Context, uuid.UUID) error { return nil }
func (f *fakeSeasonRepo) GetCropSeasonRuleByID(context.Context, uuid.UUID) (models.CropSeasonRule, error) {
	return models.CropSeasonRule{}, nil
}
func (f *fakeSeasonRepo) GetCropSeasonRuleForCrop(context.Context, uuid.UUID, *uuid.UUID) (models.CropSeasonRule, error) {
	return models.CropSeasonRule{}, nil
}
func (f *fakeSeasonRepo) GetEffectiveSeasonForCrop(_ context.Context, crop, farm uuid.UUID) (models.Season, bool, error) {
	byFarm, ok := f.seasonByCropAndFarm[crop]
	if !ok {
		return models.Season{}, false, nil
	}
	season, ok := byFarm[farm]
	if !ok {
		return models.Season{}, false, nil
	}
	return season, true, nil
}
func (f *fakeSeasonRepo) GetEffectiveSeasonsForCrops(context.Context, []models.CropAtFarm) (map[models.CropAtFarm]models.Season, error) {
	return nil, nil
}

func newTestRentalService(rentalRepo *fakeRentalRepo, plotRepo *fakePlotRepo, cropRepo *fakeCropRepo, farmRepo *fakeFarmRepo, fieldRepo *fakeFieldRepo) RentalService {
	return newTestRentalServiceWithSeasons(rentalRepo, plotRepo, cropRepo, farmRepo, fieldRepo, &fakeSeasonRepo{})
}

func newTestRentalServiceWithSeasons(rentalRepo *fakeRentalRepo, plotRepo *fakePlotRepo, cropRepo *fakeCropRepo, farmRepo *fakeFarmRepo, fieldRepo *fakeFieldRepo, seasonRepo *fakeSeasonRepo) RentalService {
	return NewRentalService(farmRepo, fieldRepo, rentalRepo, plotRepo, cropRepo, seasonRepo)
}

func TestRentalService_RequestRental_RejectsStartDateOutsideWindow(t *testing.T) {
	plot := uuid.New()
	crop := uuid.New()
	plotRepo := &fakePlotRepo{fieldByPlot: map[uuid.UUID]uuid.UUID{plot: uuid.New()}}
	cropRepo := &fakeCropRepo{crop: models.Crop{ID: crop, DurationMonths: 6}, offeredCropIDs: []uuid.UUID{crop}}
	svc := newTestRentalService(&fakeRentalRepo{}, plotRepo, cropRepo, &fakeFarmRepo{}, &fakeFieldRepo{})

	cases := []struct {
		name    string
		startAt time.Time
	}{
		{"too soon", time.Now().Add(1 * time.Hour)},
		{"too far out", time.Now().Add(61 * 24 * time.Hour)},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := svc.RequestRental(context.Background(), uuid.New(), plot, crop, tc.startAt, "please")
			if !errors.Is(err, ErrInvalidRentalRequest) {
				t.Fatalf("error = %v, want ErrInvalidRentalRequest", err)
			}
		})
	}
}

func TestRentalService_RequestRental_RejectsBlankMessage(t *testing.T) {
	plot := uuid.New()
	crop := uuid.New()
	plotRepo := &fakePlotRepo{fieldByPlot: map[uuid.UUID]uuid.UUID{plot: uuid.New()}}
	cropRepo := &fakeCropRepo{crop: models.Crop{ID: crop, DurationMonths: 6}, offeredCropIDs: []uuid.UUID{crop}}
	svc := newTestRentalService(&fakeRentalRepo{}, plotRepo, cropRepo, &fakeFarmRepo{}, &fakeFieldRepo{})

	_, err := svc.RequestRental(context.Background(), uuid.New(), plot, crop, time.Now().Add(24*time.Hour), "   ")
	if !errors.Is(err, ErrInvalidRentalRequest) {
		t.Fatalf("error = %v, want ErrInvalidRentalRequest", err)
	}
}

func TestRentalService_RequestRental_OK(t *testing.T) {
	plot := uuid.New()
	field := uuid.New()
	crop := uuid.New()
	customer := uuid.New()
	plotRepo := &fakePlotRepo{fieldByPlot: map[uuid.UUID]uuid.UUID{plot: field}}
	cropRepo := &fakeCropRepo{crop: models.Crop{ID: crop, DurationMonths: 6}, offeredCropIDs: []uuid.UUID{crop}}
	fieldRepo := &fakeFieldRepo{farmByField: map[uuid.UUID]uuid.UUID{field: uuid.New()}}
	svc := newTestRentalService(&fakeRentalRepo{}, plotRepo, cropRepo, &fakeFarmRepo{}, fieldRepo)

	startAt := time.Now().Add(10 * 24 * time.Hour)
	rental, err := svc.RequestRental(context.Background(), customer, plot, crop, startAt, "please let me rent this")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if rental.Status != models.RentalStatusRequested {
		t.Errorf("status = %v, want %v", rental.Status, models.RentalStatusRequested)
	}
	if rental.Message != "please let me rent this" {
		t.Errorf("message = %q, want the submitted message", rental.Message)
	}
}

func TestRentalService_ApproveRental_ForbiddenWhenFarmerDoesNotOwnPlot(t *testing.T) {
	rentalID := uuid.New()
	field := uuid.New()
	farmer := uuid.New()

	rentalRepo := &fakeRentalRepo{
		rentalByID: map[uuid.UUID]models.RentalWithField{
			rentalID: {Rental: models.Rental{ID: rentalID, Status: models.RentalStatusRequested}, Field: field},
		},
	}
	farmRepo := &fakeFarmRepo{farmIDByFarmer: map[uuid.UUID]uuid.UUID{farmer: uuid.New()}}
	fieldRepo := &fakeFieldRepo{farmByField: map[uuid.UUID]uuid.UUID{field: uuid.New()}}

	svc := newTestRentalService(rentalRepo, &fakePlotRepo{}, &fakeCropRepo{}, farmRepo, fieldRepo)

	_, err := svc.ApproveRental(context.Background(), farmer, rentalID)
	if !errors.Is(err, ErrForbidden) {
		t.Fatalf("error = %v, want ErrForbidden", err)
	}
}

func TestRentalService_ApproveRental_OK(t *testing.T) {
	rentalID := uuid.New()
	field := uuid.New()
	farm := uuid.New()
	farmer := uuid.New()

	rentalRepo := &fakeRentalRepo{
		rentalByID: map[uuid.UUID]models.RentalWithField{
			rentalID: {Rental: models.Rental{ID: rentalID, Status: models.RentalStatusRequested}, Field: field},
		},
		updated: models.Rental{ID: rentalID, Status: models.RentalStatusApproved},
	}
	farmRepo := &fakeFarmRepo{farmIDByFarmer: map[uuid.UUID]uuid.UUID{farmer: farm}}
	fieldRepo := &fakeFieldRepo{farmByField: map[uuid.UUID]uuid.UUID{field: farm}}

	svc := newTestRentalService(rentalRepo, &fakePlotRepo{}, &fakeCropRepo{}, farmRepo, fieldRepo)

	rental, err := svc.ApproveRental(context.Background(), farmer, rentalID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if rental.Status != models.RentalStatusApproved {
		t.Errorf("status = %v, want %v", rental.Status, models.RentalStatusApproved)
	}
}

func TestRentalService_RequestRental_Season(t *testing.T) {
	// startAt must also satisfy the 1-60 day notice window (see
	// minRentalNotice/maxRentalNotice), so it is fixed relative to "now" and
	// the season is built around it rather than the other way around.
	startAt := time.Now().AddDate(0, 0, 10)

	containingSeason := models.Season{
		Name:       "Test-Season",
		StartMonth: int32(startAt.AddDate(0, 0, -3).Month()), StartDay: int32(startAt.AddDate(0, 0, -3).Day()),
		EndMonth: int32(startAt.AddDate(0, 1, 3).Month()), EndDay: int32(startAt.AddDate(0, 1, 3).Day()),
	}
	laterSeason := models.Season{
		Name:       "Test-Season",
		StartMonth: int32(startAt.AddDate(0, 0, 5).Month()), StartDay: int32(startAt.AddDate(0, 0, 5).Day()),
		EndMonth: int32(startAt.AddDate(0, 1, 5).Month()), EndDay: int32(startAt.AddDate(0, 1, 5).Day()),
	}

	cases := []struct {
		name    string
		season  *models.Season // nil means the crop has no season rule at all: unrestricted
		wantErr error
	}{
		{name: "no rule on crop is unaffected"},
		{name: "fits inside season", season: &containingSeason},
		{name: "starts before season", season: &laterSeason, wantErr: ErrOutsideSeason},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			plot := uuid.New()
			field := uuid.New()
			farm := uuid.New()
			cropID := uuid.New()

			crop := models.Crop{ID: cropID, DurationMonths: 1}

			plotRepo := &fakePlotRepo{fieldByPlot: map[uuid.UUID]uuid.UUID{plot: field}}
			fieldRepo := &fakeFieldRepo{farmByField: map[uuid.UUID]uuid.UUID{field: farm}}
			cropRepo := &fakeCropRepo{crop: crop, offeredCropIDs: []uuid.UUID{cropID}}

			seasonRepo := &fakeSeasonRepo{}
			if tc.season != nil {
				seasonRepo.seasonByCropAndFarm = map[uuid.UUID]map[uuid.UUID]models.Season{
					cropID: {farm: *tc.season},
				}
			}

			svc := newTestRentalServiceWithSeasons(&fakeRentalRepo{}, plotRepo, cropRepo, &fakeFarmRepo{}, fieldRepo, seasonRepo)

			_, err := svc.RequestRental(context.Background(), uuid.New(), plot, cropID, startAt, "please")
			if tc.wantErr == nil {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				return
			}
			if !errors.Is(err, tc.wantErr) {
				t.Fatalf("error = %v, want %v", err, tc.wantErr)
			}
		})
	}
}

func TestSeasonContainsPeriod_Wraparound(t *testing.T) {
	winter := models.Season{StartMonth: 12, StartDay: 1, EndMonth: 2, EndDay: 28}

	cases := []struct {
		name    string
		startAt time.Time
		endAt   time.Time
		want    bool
	}{
		{
			name:    "rental crossing new year fits inside wraparound season",
			startAt: time.Date(2026, time.December, 20, 0, 0, 0, 0, time.UTC),
			endAt:   time.Date(2027, time.January, 20, 0, 0, 0, 0, time.UTC),
			want:    true,
		},
		{
			name:    "rental just outside wraparound season",
			startAt: time.Date(2027, time.March, 1, 0, 0, 0, 0, time.UTC),
			endAt:   time.Date(2027, time.March, 15, 0, 0, 0, 0, time.UTC),
			want:    false,
		},
		{
			name:    "rental entirely in the january fragment",
			startAt: time.Date(2027, time.January, 5, 0, 0, 0, 0, time.UTC),
			endAt:   time.Date(2027, time.February, 1, 0, 0, 0, 0, time.UTC),
			want:    true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := seasonContainsPeriod(winter, tc.startAt, tc.endAt)
			if got != tc.want {
				t.Errorf("seasonContainsPeriod() = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestRentalService_DeclineRental_AlreadyDecided(t *testing.T) {
	rentalID := uuid.New()
	field := uuid.New()
	farm := uuid.New()
	farmer := uuid.New()

	rentalRepo := &fakeRentalRepo{
		rentalByID: map[uuid.UUID]models.RentalWithField{
			rentalID: {Rental: models.Rental{ID: rentalID, Status: models.RentalStatusApproved}, Field: field},
		},
		updateErr: ErrRentalAlreadyDecided,
	}
	farmRepo := &fakeFarmRepo{farmIDByFarmer: map[uuid.UUID]uuid.UUID{farmer: farm}}
	fieldRepo := &fakeFieldRepo{farmByField: map[uuid.UUID]uuid.UUID{field: farm}}

	svc := newTestRentalService(rentalRepo, &fakePlotRepo{}, &fakeCropRepo{}, farmRepo, fieldRepo)

	_, err := svc.DeclineRental(context.Background(), farmer, rentalID)
	if !errors.Is(err, ErrRentalAlreadyDecided) {
		t.Fatalf("error = %v, want ErrRentalAlreadyDecided", err)
	}
}
