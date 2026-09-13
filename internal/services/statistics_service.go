package services

import (
	"context"
	"fmt"

	"github.com/Neue-Konzepte-BaaS/backend/internal/models"
	"github.com/google/uuid"
)

type StatisticsService interface {
	// GetStatistics returns what the account is allowed to see: a farmer gets
	// their own holdings, an admin the platform totals. Any other role gets
	// ErrForbidden.
	GetStatistics(ctx context.Context, account uuid.UUID, role models.Role) (models.Statistics, error)
}

type statisticsService struct {
	statisticsRepo StatisticsRepository
}

func NewStatisticsService(statisticsRepo StatisticsRepository) StatisticsService {
	return &statisticsService{statisticsRepo: statisticsRepo}
}

func (s *statisticsService) GetStatistics(ctx context.Context, account uuid.UUID, role models.Role) (models.Statistics, error) {
	switch role {
	case models.RoleFarmer:
		stats, err := s.statisticsRepo.GetFarmStatistics(ctx, account)
		if err != nil {
			return models.Statistics{}, fmt.Errorf("getting farm statistics: %w", err)
		}
		stats.Scope = models.ScopeFarm
		return withDerivedStatistics(stats), nil
	case models.RoleAdmin:
		stats, err := s.statisticsRepo.GetPlatformStatistics(ctx)
		if err != nil {
			return models.Statistics{}, fmt.Errorf("getting platform statistics: %w", err)
		}
		stats.Scope = models.ScopePlatform
		return withDerivedStatistics(stats), nil
	default:
		// The route is gated by RequireAnyRole(farmer, admin), so this is
		// defence in depth -- the same belt-and-braces as plotService's
		// ownership check behind RequireRole(farmer).
		return models.Statistics{}, ErrForbidden
	}
}

// withDerivedStatistics fills the figures computed from the measured ones.
// Kept in Go, not SQL, so a divide-by-zero for a farmer with no plots yet is
// an ordinary guarded branch rather than a NULL crossing into pgx.
func withDerivedStatistics(stats models.Statistics) models.Statistics {
	stats.Plots.Available = stats.Plots.Total - stats.Plots.Rented
	if stats.Plots.Total > 0 {
		stats.Plots.OccupancyRate = float64(stats.Plots.Rented) / float64(stats.Plots.Total)
	}
	return stats
}
