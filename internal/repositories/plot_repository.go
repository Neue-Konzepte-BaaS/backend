package repositories

import (
	"context"

	"github.com/Neue-Konzepte-BaaS/backend/internal/models"
	database "github.com/Neue-Konzepte-BaaS/backend/internal/repositories/db"
	"github.com/Neue-Konzepte-BaaS/backend/internal/services"
	"github.com/google/uuid"
)

type plotRepository struct {
	queries *database.Queries
}

func NewPlotRepository(queries *database.Queries) services.PlotRepository {
	return &plotRepository{queries: queries}
}

func (r *plotRepository) CreatePlot(ctx context.Context, plot models.Plot) (uuid.UUID, error) {
	id, err := r.queries.InsertPlot(ctx, database.InsertPlotParams{
		Name:        plot.Name,
		Field:       plot.Field,
		Coordinates: plot.Coordinates,
	})
	if err != nil {
		return uuid.UUID{}, mapGeometryError(err)
	}
	return id, nil
}

func (r *plotRepository) GetPlotsByFields(ctx context.Context, fields []uuid.UUID) ([]models.Plot, error) {
	rows, err := r.queries.GetPlotsByFields(ctx, fields)
	if err != nil {
		return nil, err
	}

	plots := make([]models.Plot, len(rows))
	for i, row := range rows {
		plots[i] = models.Plot{
			ID:          row.ID,
			Name:        row.Name,
			Field:       row.Field,
			Coordinates: row.Coordinates,
		}
	}
	return plots, nil
}
