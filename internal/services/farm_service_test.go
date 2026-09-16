package services

import (
	"context"
	"errors"
	"testing"

	"github.com/Neue-Konzepte-BaaS/backend/internal/models"
	"github.com/google/uuid"
)

type fakeFarmRepo struct {
	page   models.Page[models.FarmListing]
	err    error
	called bool
	filter models.FarmListFilter
}

func (f *fakeFarmRepo) ListFarms(_ context.Context, filter models.FarmListFilter) (models.Page[models.FarmListing], error) {
	f.called = true
	f.filter = filter
	if f.err != nil {
		return models.Page[models.FarmListing]{}, f.err
	}
	return f.page, nil
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
			if repo.called != tt.wantCalls {
				t.Errorf("repository called = %v, want %v", repo.called, tt.wantCalls)
			}
		})
	}
}

// A farmer must not reach the farm list even to see his own farm: the scope
// here is the platform, and there is no per-farmer variant of it.
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

	statsRepo := &fakeStatisticsRepo{farmStats: models.Statistics{Plots: plots}}
	fromStats, err := NewStatisticsService(statsRepo).GetStatistics(context.Background(), uuid.Nil, models.RoleFarmer)
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

	if repo.filter.Query != "green acres" {
		t.Errorf("query = %q, want %q", repo.filter.Query, "green acres")
	}
	if repo.filter.Limit != MaxPageLimit {
		t.Errorf("limit = %d, want %d", repo.filter.Limit, MaxPageLimit)
	}
	if repo.filter.Offset != 0 {
		t.Errorf("offset = %d, want 0", repo.filter.Offset)
	}
}

func TestListFarms_PassesThePostalCodeFilterThrough(t *testing.T) {
	code := int32(76133)
	repo := &fakeFarmRepo{}
	svc := NewFarmService(repo)

	if _, err := svc.ListFarms(context.Background(), models.RoleAdmin, models.FarmListFilter{PostalCode: &code}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if repo.filter.PostalCode == nil || *repo.filter.PostalCode != code {
		t.Errorf("postal code = %v, want %d", repo.filter.PostalCode, code)
	}
}

func TestListFarms_WrapsRepositoryErrors(t *testing.T) {
	sentinel := errors.New("connection refused")
	repo := &fakeFarmRepo{err: sentinel}
	svc := NewFarmService(repo)

	_, err := svc.ListFarms(context.Background(), models.RoleAdmin, models.FarmListFilter{})
	if !errors.Is(err, sentinel) {
		t.Fatalf("error = %v, want it to wrap %v", err, sentinel)
	}
}
