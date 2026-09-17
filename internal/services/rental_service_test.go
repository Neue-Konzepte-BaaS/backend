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
	created    models.Rental
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

// fakePlotRepo is an in-memory PlotRepository, only implementing what
// rentalService needs.
type fakePlotRepo struct {
	fieldByPlot map[uuid.UUID]uuid.UUID
}

func (f *fakePlotRepo) CreatePlot(context.Context, models.Plot) (models.Plot, error) {
	return models.Plot{}, nil
}

func (f *fakePlotRepo) GetPlotsByFields(context.Context, []uuid.UUID) ([]models.Plot, error) {
	return nil, nil
}

func (f *fakePlotRepo) GetNearestPlots(context.Context, float64, float64, int32) ([]models.NearbyPlot, error) {
	return nil, nil
}

func (f *fakePlotRepo) GetPlotField(_ context.Context, plot uuid.UUID) (uuid.UUID, error) {
	if field, ok := f.fieldByPlot[plot]; ok {
		return field, nil
	}
	return uuid.UUID{}, ErrNotFound
}

// fakeFieldRepo is an in-memory FieldRepository, only implementing what
// rentalService needs.
type fakeFieldRepo struct {
	farmByField map[uuid.UUID]uuid.UUID
}

func (f *fakeFieldRepo) CreateField(context.Context, models.Field) (uuid.UUID, error) {
	return uuid.UUID{}, nil
}

func (f *fakeFieldRepo) GetFieldFarm(_ context.Context, id uuid.UUID) (uuid.UUID, error) {
	if farm, ok := f.farmByField[id]; ok {
		return farm, nil
	}
	return uuid.UUID{}, ErrNotFound
}

func (f *fakeFieldRepo) GetFieldsByFarm(context.Context, uuid.UUID) ([]models.Field, error) {
	return nil, nil
}

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
func (f *fakeCropRepo) SetPlotCrops(context.Context, uuid.UUID, []uuid.UUID) error { return nil }
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

func newTestRentalService(rentalRepo *fakeRentalRepo, plotRepo *fakePlotRepo, cropRepo *fakeCropRepo, farmRepo *fakeFarmRepo, fieldRepo *fakeFieldRepo) RentalService {
	return NewRentalService(farmRepo, fieldRepo, rentalRepo, plotRepo, cropRepo)
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
	crop := uuid.New()
	customer := uuid.New()
	plotRepo := &fakePlotRepo{fieldByPlot: map[uuid.UUID]uuid.UUID{plot: uuid.New()}}
	cropRepo := &fakeCropRepo{crop: models.Crop{ID: crop, DurationMonths: 6}, offeredCropIDs: []uuid.UUID{crop}}
	svc := newTestRentalService(&fakeRentalRepo{}, plotRepo, cropRepo, &fakeFarmRepo{}, &fakeFieldRepo{})

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
