package repositories

import (
	"context"

	"github.com/Neue-Konzepte-BaaS/backend/internal/models"
	database "github.com/Neue-Konzepte-BaaS/backend/internal/repositories/db"
	"github.com/Neue-Konzepte-BaaS/backend/internal/services"
	"github.com/google/uuid"
)

type statisticsRepository struct {
	queries *database.Queries
}

// NewStatisticsRepository has no error mapping to do, unlike its siblings:
// both queries are single-row aggregates that touch no constraint and always
// return exactly one row, so there is no SQLSTATE to translate and no
// pgx.ErrNoRows case.
func NewStatisticsRepository(queries *database.Queries) services.StatisticsRepository {
	return &statisticsRepository{queries: queries}
}

func (r *statisticsRepository) GetFarmStatistics(ctx context.Context, farmer uuid.UUID) (models.Statistics, error) {
	row, err := r.queries.GetFarmStatistics(ctx, farmer)
	if err != nil {
		return models.Statistics{}, err
	}
	return models.Statistics{
		GeneratedAt: row.GeneratedAt.Time,
		Fields: models.FieldStatistics{
			Total:            row.FieldCount,
			AreaSquareMeters: row.FieldAreaSquareMeters,
		},
		Plots: models.PlotStatistics{
			Total:            row.PlotCount,
			Rented:           row.RentedPlotCount,
			AreaSquareMeters: row.PlotAreaSquareMeters,
		},
		Rentals: models.RentalStatistics{
			Total:      row.RentalCount,
			Active:     row.ActiveRentalCount,
			Last30Days: row.RentalsLast30Days,
		},
	}, nil
}

func (r *statisticsRepository) GetPlatformStatistics(ctx context.Context) (models.Statistics, error) {
	row, err := r.queries.GetPlatformStatistics(ctx)
	if err != nil {
		return models.Statistics{}, err
	}
	return models.Statistics{
		GeneratedAt: row.GeneratedAt.Time,
		Fields: models.FieldStatistics{
			Total:            row.FieldCount,
			AreaSquareMeters: row.FieldAreaSquareMeters,
		},
		Plots: models.PlotStatistics{
			Total:            row.PlotCount,
			Rented:           row.RentedPlotCount,
			AreaSquareMeters: row.PlotAreaSquareMeters,
		},
		Rentals: models.RentalStatistics{
			Total:      row.RentalCount,
			Active:     row.ActiveRentalCount,
			Last30Days: row.RentalsLast30Days,
		},
		Accounts: &models.AccountStatistics{
			Total:                row.AccountCount,
			Farmers:              row.FarmerCount,
			Customers:            row.CustomerCount,
			RegisteredLast30Days: row.AccountsLast30Days,
		},
	}, nil
}
