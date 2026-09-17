package services

import (
	"context"
	"errors"
	"testing"

	"github.com/Neue-Konzepte-BaaS/backend/internal/models"
	"github.com/google/uuid"
)

// fakeFarmRepo is an in-memory FarmRepository for exercising the service
// layer without a database. farmIDByFarmer maps a farmer's account id to
// their farm id, for GetFarmIDByFarmerID. The listing fields record what the
// service passed down, so tests can prove that and not only what came back.
type fakeFarmRepo struct {
	farm           models.Farm
	farmErr        error
	farmIDByFarmer map[uuid.UUID]uuid.UUID
	farmIDErr      error

	page       models.Page[models.FarmListing]
	listErr    error
	listCalled bool
	listFilter models.FarmListFilter
}

func (f *fakeFarmRepo) ListFarms(_ context.Context, filter models.FarmListFilter) (models.Page[models.FarmListing], error) {
	f.listCalled = true
	f.listFilter = filter
	if f.listErr != nil {
		return models.Page[models.FarmListing]{}, f.listErr
	}
	return f.page, nil
}

func (f *fakeFarmRepo) GetFarmByID(context.Context, uuid.UUID) (models.Farm, error) {
	return f.farm, f.farmErr
}

func (f *fakeFarmRepo) GetFarmIDByFarmerID(_ context.Context, farmerID uuid.UUID) (uuid.UUID, error) {
	if f.farmIDErr != nil {
		return uuid.UUID{}, f.farmIDErr
	}
	if id, ok := f.farmIDByFarmer[farmerID]; ok {
		return id, nil
	}
	return uuid.UUID{}, ErrNotFound
}

func TestFarmService_GetFarm_OK(t *testing.T) {
	farmID := uuid.New()
	want := models.Farm{
		ID:                farmID,
		FarmerID:          uuid.New(),
		Name:              "Green Acres",
		Address:           "1 Farm Lane",
		Description:       "A small family farm",
		TotalSquareMeters: 1234.5,
	}
	svc := NewFarmService(&fakeFarmRepo{farm: want})

	got, err := svc.GetFarm(context.Background(), farmID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != want {
		t.Errorf("farm = %+v, want %+v", got, want)
	}
}

func TestFarmService_GetFarm_NotFoundPropagates(t *testing.T) {
	svc := NewFarmService(&fakeFarmRepo{farmErr: ErrNotFound})

	_, err := svc.GetFarm(context.Background(), uuid.New())
	if !errors.Is(err, ErrNotFound) {
		t.Errorf("err = %v, want ErrNotFound", err)
	}
}

func TestListFarms_OnlyAdminReachesTheRepository(t *testing.T) {
	tests := []struct {
		name      string
		role      models.Role
		wantErr   error
		wantCalls bool
	}{
		{name: "admin", role: models.RoleAdmin, wantCalls: true},
		{name: "farmer", role: models.RoleFarmer, wantErr: ErrForbidden},
		{name: "customer", role: models.RoleCustomer, wantErr: ErrForbidden},
		{name: "no role at all", role: "", wantErr: ErrForbidden},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := &fakeFarmRepo{}
			svc := NewFarmService(repo)

			_, err := svc.ListFarms(context.Background(), tt.role, models.FarmListFilter{})

			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("error = %v, want %v", err, tt.wantErr)
			}
			if repo.listCalled != tt.wantCalls {
				t.Errorf("repository called = %v, want %v", repo.listCalled, tt.wantCalls)
			}
		})
	}
}

// GetFarm is public, ListFarms is not: a farmer may read any farm's public
// details but may not enumerate the platform, not even to find his own.
func TestListFarms_FarmerCannotListFarms(t *testing.T) {
	repo := &fakeFarmRepo{page: models.Page[models.FarmListing]{Total: 3}}
	svc := NewFarmService(repo)

	page, err := svc.ListFarms(context.Background(), models.RoleFarmer, models.FarmListFilter{})
	if !errors.Is(err, ErrForbidden) {
		t.Fatalf("error = %v, want %v", err, ErrForbidden)
	}
	if page.Total != 0 || page.Items != nil {
		t.Errorf("a forbidden call must return the zero page, got %+v", page)
	}
}

func TestListFarms_DerivesPlotFiguresPerFarm(t *testing.T) {
	tests := []struct {
		name              string
		total, rented     int64
		wantAvailable     int64
		wantOccupancyRate float64
	}{
		{name: "partly rented", total: 4, rented: 1, wantAvailable: 3, wantOccupancyRate: 0.25},
		{name: "fully rented", total: 2, rented: 2, wantAvailable: 0, wantOccupancyRate: 1},
		{name: "none rented", total: 2, rented: 0, wantAvailable: 2, wantOccupancyRate: 0},
		// The guarded branch: a farm with no plots must not divide by zero.
		{name: "a farm with no plots at all", total: 0, rented: 0, wantAvailable: 0, wantOccupancyRate: 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := &fakeFarmRepo{page: models.Page[models.FarmListing]{
				Items: []models.FarmListing{{
					Plots: models.PlotStatistics{Total: tt.total, Rented: tt.rented},
				}},
			}}
			svc := NewFarmService(repo)

			page, err := svc.ListFarms(context.Background(), models.RoleAdmin, models.FarmListFilter{})
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			got := page.Items[0].Plots
			if got.Available != tt.wantAvailable {
				t.Errorf("available = %d, want %d", got.Available, tt.wantAvailable)
			}
			if got.OccupancyRate != tt.wantOccupancyRate {
				t.Errorf("occupancyRate = %v, want %v", got.OccupancyRate, tt.wantOccupancyRate)
			}
		})
	}
}

// The farm list and the statistics endpoint must not develop two different
// ideas of what occupancy means, so they derive it with the same function.
func TestListFarms_DerivesOccupancyTheSameWayAsStatistics(t *testing.T) {
	plots := models.PlotStatistics{Total: 7, Rented: 3}

	repo := &fakeFarmRepo{page: models.Page[models.FarmListing]{
		Items: []models.FarmListing{{Plots: plots}},
	}}
	fromList, err := NewFarmService(repo).ListFarms(context.Background(), models.RoleAdmin, models.FarmListFilter{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	farmer := uuid.New()
	statsFarmRepo := &fakeFarmRepo{farmIDByFarmer: map[uuid.UUID]uuid.UUID{farmer: uuid.New()}}
	statsRepo := &fakeStatisticsRepo{farmStats: models.Statistics{Plots: plots}}
	fromStats, err := NewStatisticsService(statsFarmRepo, statsRepo).GetStatistics(context.Background(), farmer, models.RoleFarmer)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if fromList.Items[0].Plots != fromStats.Plots {
		t.Errorf("farm list plots = %+v, statistics plots = %+v; the two must agree", fromList.Items[0].Plots, fromStats.Plots)
	}
}

func TestListFarms_ClampsPaginationAndTrimsTheSearchTerm(t *testing.T) {
	repo := &fakeFarmRepo{}
	svc := NewFarmService(repo)

	_, err := svc.ListFarms(context.Background(), models.RoleAdmin, models.FarmListFilter{
		Query:  "  green acres  ",
		Limit:  5000,
		Offset: -3,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if repo.listFilter.Query != "green acres" {
		t.Errorf("query = %q, want %q", repo.listFilter.Query, "green acres")
	}
	if repo.listFilter.Limit != MaxPageLimit {
		t.Errorf("limit = %d, want %d", repo.listFilter.Limit, MaxPageLimit)
	}
	if repo.listFilter.Offset != 0 {
		t.Errorf("offset = %d, want 0", repo.listFilter.Offset)
	}
}

func TestListFarms_PassesThePostalCodeFilterThrough(t *testing.T) {
	code := int32(76133)
	repo := &fakeFarmRepo{}
	svc := NewFarmService(repo)

	if _, err := svc.ListFarms(context.Background(), models.RoleAdmin, models.FarmListFilter{PostalCode: &code}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if repo.listFilter.PostalCode == nil || *repo.listFilter.PostalCode != code {
		t.Errorf("postal code = %v, want %d", repo.listFilter.PostalCode, code)
	}
}

func TestListFarms_WrapsRepositoryErrors(t *testing.T) {
	sentinel := errors.New("connection refused")
	repo := &fakeFarmRepo{listErr: sentinel}
	svc := NewFarmService(repo)

	_, err := svc.ListFarms(context.Background(), models.RoleAdmin, models.FarmListFilter{})
	if !errors.Is(err, sentinel) {
		t.Fatalf("error = %v, want it to wrap %v", err, sentinel)
	}
}
