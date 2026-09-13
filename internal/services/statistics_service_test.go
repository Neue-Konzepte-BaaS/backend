package services

import (
	"context"
	"errors"
	"testing"

	"github.com/Neue-Konzepte-BaaS/backend/internal/models"
	"github.com/google/uuid"
)

// fakeStatisticsRepo is an in-memory StatisticsRepository for exercising the
// service layer without a database. It records which method was called and
// with what farmer, so tests can prove the right scope was requested.
type fakeStatisticsRepo struct {
	farmStats     models.Statistics
	platformStats models.Statistics
	farmErr       error
	platformErr   error

	calledFarmWith uuid.UUID
	farmCalled     bool
	platformCalled bool
}

func (f *fakeStatisticsRepo) GetFarmStatistics(_ context.Context, farmer uuid.UUID) (models.Statistics, error) {
	f.farmCalled = true
	f.calledFarmWith = farmer
	if f.farmErr != nil {
		return models.Statistics{}, f.farmErr
	}
	return f.farmStats, nil
}

func (f *fakeStatisticsRepo) GetPlatformStatistics(context.Context) (models.Statistics, error) {
	f.platformCalled = true
	if f.platformErr != nil {
		return models.Statistics{}, f.platformErr
	}
	return f.platformStats, nil
}

func TestGetStatistics_Farmer_UsesOwnFarmScope(t *testing.T) {
	account := uuid.New()
	repo := &fakeStatisticsRepo{farmStats: models.Statistics{Fields: models.FieldStatistics{Total: 3}}}
	svc := NewStatisticsService(repo)

	stats, err := svc.GetStatistics(context.Background(), account, models.RoleFarmer)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !repo.farmCalled {
		t.Error("expected GetFarmStatistics to be called for a farmer")
	}
	if repo.platformCalled {
		t.Error("GetPlatformStatistics must not be called for a farmer")
	}
	if repo.calledFarmWith != account {
		t.Errorf("farmer id = %v, want the caller's own id %v", repo.calledFarmWith, account)
	}
	if stats.Scope != models.ScopeFarm {
		t.Errorf("scope = %q, want %q", stats.Scope, models.ScopeFarm)
	}
	if stats.Accounts != nil {
		t.Error("a farmer's statistics must not include the accounts group")
	}
}

func TestGetStatistics_Admin_UsesPlatformScope(t *testing.T) {
	repo := &fakeStatisticsRepo{
		platformStats: models.Statistics{
			Accounts: &models.AccountStatistics{Total: 5},
		},
	}
	svc := NewStatisticsService(repo)

	stats, err := svc.GetStatistics(context.Background(), uuid.New(), models.RoleAdmin)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !repo.platformCalled {
		t.Error("expected GetPlatformStatistics to be called for an admin")
	}
	if repo.farmCalled {
		t.Error("GetFarmStatistics must not be called for an admin")
	}
	if stats.Scope != models.ScopePlatform {
		t.Errorf("scope = %q, want %q", stats.Scope, models.ScopePlatform)
	}
	if stats.Accounts == nil {
		t.Error("an admin's statistics must include the accounts group")
	}
}

func TestGetStatistics_OtherRoles_ForbiddenWithoutCallingRepo(t *testing.T) {
	for _, role := range []models.Role{models.RoleCustomer, models.Role("")} {
		t.Run(string(role), func(t *testing.T) {
			repo := &fakeStatisticsRepo{}
			svc := NewStatisticsService(repo)

			_, err := svc.GetStatistics(context.Background(), uuid.New(), role)
			if !errors.Is(err, ErrForbidden) {
				t.Fatalf("error = %v, want ErrForbidden", err)
			}
			if repo.farmCalled || repo.platformCalled {
				t.Error("repository must not be called for a role with no statistics")
			}
		})
	}
}

func TestGetStatistics_RepositoryErrorPropagates(t *testing.T) {
	sentinel := errors.New("db exploded")

	t.Run("farm", func(t *testing.T) {
		repo := &fakeStatisticsRepo{farmErr: sentinel}
		svc := NewStatisticsService(repo)

		_, err := svc.GetStatistics(context.Background(), uuid.New(), models.RoleFarmer)
		if !errors.Is(err, sentinel) {
			t.Fatalf("error = %v, want wrapped %v", err, sentinel)
		}
	})

	t.Run("platform", func(t *testing.T) {
		repo := &fakeStatisticsRepo{platformErr: sentinel}
		svc := NewStatisticsService(repo)

		_, err := svc.GetStatistics(context.Background(), uuid.New(), models.RoleAdmin)
		if !errors.Is(err, sentinel) {
			t.Fatalf("error = %v, want wrapped %v", err, sentinel)
		}
	})
}

func TestWithDerivedStatistics_OccupancyRate(t *testing.T) {
	tests := []struct {
		name          string
		total, rented int64
		wantAvailable int64
		wantRate      float64
	}{
		{name: "partial occupancy", total: 10, rented: 4, wantAvailable: 6, wantRate: 0.4},
		{name: "no plots at all", total: 0, rented: 0, wantAvailable: 0, wantRate: 0},
		{name: "fully rented", total: 3, rented: 3, wantAvailable: 0, wantRate: 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			stats := models.Statistics{Plots: models.PlotStatistics{Total: tt.total, Rented: tt.rented}}
			got := withDerivedStatistics(stats)

			if got.Plots.Available != tt.wantAvailable {
				t.Errorf("available = %d, want %d", got.Plots.Available, tt.wantAvailable)
			}
			if got.Plots.OccupancyRate != tt.wantRate {
				t.Errorf("occupancyRate = %v, want %v", got.Plots.OccupancyRate, tt.wantRate)
			}
		})
	}
}
