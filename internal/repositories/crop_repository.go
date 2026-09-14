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
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

type cropRepository struct {
	pool    *pgxpool.Pool
	queries *database.Queries
}

func NewCropRepository(pool *pgxpool.Pool, queries *database.Queries) services.CropRepository {
	return &cropRepository{pool: pool, queries: queries}
}

func (r *cropRepository) CreateCrop(ctx context.Context, name string, durationMonths int32) (models.Crop, error) {
	id, err := r.queries.InsertCrop(ctx, database.InsertCropParams{
		Name:           name,
		DurationMonths: durationMonths,
	})
	if err != nil {
		if isUniqueViolation(err) {
			return models.Crop{}, services.ErrCropNameTaken
		}
		return models.Crop{}, err
	}
	return models.Crop{ID: id, Name: name, DurationMonths: durationMonths}, nil
}

func (r *cropRepository) GetAllCrops(ctx context.Context) ([]models.Crop, error) {
	rows, err := r.queries.GetAllCrops(ctx)
	if err != nil {
		return nil, err
	}
	return toCrops(rows), nil
}

func (r *cropRepository) GetCropByID(ctx context.Context, id uuid.UUID) (models.Crop, error) {
	row, err := r.queries.GetCropByID(ctx, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return models.Crop{}, fmt.Errorf("db error: %w %w", err, services.ErrNotFound)
		}
		return models.Crop{}, err
	}
	return models.Crop{ID: row.ID, Name: row.Name, DurationMonths: row.DurationMonths}, nil
}

// SetPlotCrops replaces the plot's offered crops in a single transaction, so
// a caller never observes a partially-updated set.
func (r *cropRepository) SetPlotCrops(ctx context.Context, plot uuid.UUID, crops []uuid.UUID) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }() // no-op after a successful Commit

	qtx := r.queries.WithTx(tx)

	if err := qtx.DeletePlotCrops(ctx, plot); err != nil {
		return fmt.Errorf("clearing plot crops: %w", err)
	}

	for _, crop := range crops {
		if err := qtx.InsertPlotCrop(ctx, database.InsertPlotCropParams{Plot: plot, Crop: crop}); err != nil {
			return mapCropError(err)
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit tx: %w", err)
	}
	return nil
}

func (r *cropRepository) GetCropsByPlot(ctx context.Context, plot uuid.UUID) ([]models.Crop, error) {
	rows, err := r.queries.GetCropsByPlot(ctx, plot)
	if err != nil {
		return nil, err
	}
	return toCrops(rows), nil
}

func (r *cropRepository) GetCropsByPlots(ctx context.Context, plots []uuid.UUID) (map[uuid.UUID][]models.Crop, error) {
	rows, err := r.queries.GetCropsByPlots(ctx, plots)
	if err != nil {
		return nil, err
	}

	cropsByPlot := make(map[uuid.UUID][]models.Crop, len(plots))
	for _, row := range rows {
		cropsByPlot[row.Plot] = append(cropsByPlot[row.Plot], models.Crop{
			ID:             row.ID,
			Name:           row.Name,
			DurationMonths: row.DurationMonths,
		})
	}
	return cropsByPlot, nil
}

// mapCropError turns the foreign key violation raised by inserting an
// unknown crop id (or a plot id that no longer exists) into ErrNotFound, so
// handlers can report it as a 404 instead of leaking a raw SQL error as a 500.
func mapCropError(err error) error {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == foreignKeyViolation {
		return fmt.Errorf("%s: %w", pgErr.Message, services.ErrNotFound)
	}
	return err
}

func toCrops(rows []database.Crop) []models.Crop {
	crops := make([]models.Crop, len(rows))
	for i, row := range rows {
		crops[i] = models.Crop{ID: row.ID, Name: row.Name, DurationMonths: row.DurationMonths}
	}
	return crops
}
