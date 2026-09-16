package repositories

import (
	"context"

	"github.com/Neue-Konzepte-BaaS/backend/internal/models"
	database "github.com/Neue-Konzepte-BaaS/backend/internal/repositories/db"
	"github.com/Neue-Konzepte-BaaS/backend/internal/services"
	"github.com/jackc/pgx/v5/pgtype"
)

type farmRepository struct {
	queries *database.Queries
}

func NewFarmRepository(queries *database.Queries) services.FarmRepository {
	return &farmRepository{queries: queries}
}

func (r *farmRepository) ListFarms(ctx context.Context, filter models.FarmListFilter) (models.Page[models.FarmListing], error) {
	postalCode := pgtype.Int4{}
	if filter.PostalCode != nil {
		postalCode = pgtype.Int4{Int32: *filter.PostalCode, Valid: true}
	}

	rows, err := r.queries.ListFarms(ctx, database.ListFarmsParams{
		Search:       filter.Query,
		PostalCode:   postalCode,
		ResultLimit:  filter.Limit,
		ResultOffset: filter.Offset,
	})
	if err != nil {
		return models.Page[models.FarmListing]{}, err
	}

	page := models.Page[models.FarmListing]{
		Items:  make([]models.FarmListing, len(rows)),
		Limit:  filter.Limit,
		Offset: filter.Offset,
	}
	for i, row := range rows {
		page.Items[i] = toModelFarmListing(row)
	}
	// Every row carries the same window-function count; see ListAccounts.
	if len(rows) > 0 {
		page.Total = rows[0].TotalCount
	}
	return page, nil
}

// toModelFarmListing leaves Plots.Available and Plots.OccupancyRate zero: they
// are derived, and deriving them is the service's job.
func toModelFarmListing(row database.ListFarmsRow) models.FarmListing {
	return models.FarmListing{
		Account:    row.ID,
		FarmName:   row.FarmName,
		PostalCode: row.PostalCode,
		FirstName:  row.FirstName,
		LastName:   row.LastName,
		Email:      row.Email,
		CreatedAt:  row.CreatedAt.Time,
		Fields: models.FieldStatistics{
			Total:            row.FieldCount,
			AreaSquareMeters: row.FieldAreaSquareMeters,
		},
		Plots: models.PlotStatistics{
			Total:            row.PlotCount,
			Rented:           row.RentedPlotCount,
			AreaSquareMeters: row.PlotAreaSquareMeters,
		},
		ActiveRentals: row.ActiveRentalCount,
	}
}
