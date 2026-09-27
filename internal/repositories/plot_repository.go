package repositories

import (
	"context"
	"errors"
	"fmt"

	"github.com/Neue-Konzepte-BaaS/backend/internal/models"
	database "github.com/Neue-Konzepte-BaaS/backend/internal/repositories/db"
	"github.com/Neue-Konzepte-BaaS/backend/internal/services"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

type plotRepository struct {
	queries *database.Queries
}

func NewPlotRepository(queries *database.Queries) services.PlotRepository {
	return &plotRepository{queries: queries}
}

func (r *plotRepository) CreatePlot(ctx context.Context, plot models.Plot) (models.Plot, error) {
	row, err := r.queries.InsertPlot(ctx, database.InsertPlotParams{
		Name:        plot.Name,
		Field:       plot.Field,
		Coordinates: plot.Coordinates,
	})
	if err != nil {
		return models.Plot{}, mapGeometryError(err)
	}
	plot.ID = row.ID
	plot.AreaSquareMeters = row.AreaSquareMeters
	return plot, nil
}

func (r *plotRepository) GetPlotsByFields(ctx context.Context, fields []uuid.UUID) ([]models.Plot, error) {
	rows, err := r.queries.GetPlotsByFields(ctx, fields)
	if err != nil {
		return nil, err
	}

	plots := make([]models.Plot, len(rows))
	for i, row := range rows {
		plots[i] = models.Plot{
			ID:                          row.ID,
			Name:                        row.Name,
			Field:                       row.Field,
			Coordinates:                 row.Coordinates,
			AreaSquareMeters:            row.AreaSquareMeters,
			BasePriceCentsPerSqmPerWeek: fromPgInt4(row.BasePriceCentsPerSqmPerWeek),
		}
	}
	return plots, nil
}

func (r *plotRepository) GetPlotField(ctx context.Context, plot uuid.UUID) (uuid.UUID, error) {
	field, err := r.queries.GetPlotField(ctx, plot)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return uuid.UUID{}, fmt.Errorf("db error: %w %w", err, services.ErrNotFound)
		}
		return uuid.UUID{}, err
	}
	return field, nil
}

func (r *plotRepository) GetPlotByID(ctx context.Context, plot uuid.UUID) (models.Plot, error) {
	row, err := r.queries.GetPlotByID(ctx, plot)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return models.Plot{}, fmt.Errorf("db error: %w %w", err, services.ErrNotFound)
		}
		return models.Plot{}, err
	}
	return models.Plot{
		ID:                          row.ID,
		Name:                        row.Name,
		Field:                       row.Field,
		Coordinates:                 row.Coordinates,
		AreaSquareMeters:            row.AreaSquareMeters,
		BasePriceCentsPerSqmPerWeek: fromPgInt4(row.BasePriceCentsPerSqmPerWeek),
	}, nil
}

// fromPgInt4 converts a nullable Postgres int4 to a nullable Go int32.
func fromPgInt4(v pgtype.Int4) *int32 {
	if !v.Valid {
		return nil
	}
	value := v.Int32
	return &value
}

func (r *plotRepository) GetNearestPlots(ctx context.Context, lon, lat float64, farm *uuid.UUID, limit int32) ([]models.NearbyPlot, error) {
	rows, err := r.queries.GetNearestPlots(ctx, database.GetNearestPlotsParams{
		Lon:         lon,
		Lat:         lat,
		Farm:        farm,
		ResultLimit: limit,
	})
	if err != nil {
		return nil, err
	}

	plots := make([]models.NearbyPlot, len(rows))
	for i, row := range rows {
		plots[i] = models.NearbyPlot{
			Plot: models.Plot{
				ID:               row.ID,
				Name:             row.Name,
				Field:            row.Field,
				Coordinates:      row.Coordinates,
				AreaSquareMeters: row.AreaSquareMeters,
			},
			Farm:           row.Farm,
			DistanceMeters: row.DistanceMeters,
		}
	}
	return plots, nil
}
