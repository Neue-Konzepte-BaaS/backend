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
	"github.com/jackc/pgx/v5/pgxpool"
)

type seasonRepository struct {
	pool    *pgxpool.Pool
	queries *database.Queries
}

func NewSeasonRepository(pool *pgxpool.Pool, queries *database.Queries) services.SeasonRepository {
	return &seasonRepository{pool: pool, queries: queries}
}

func (r *seasonRepository) CreateSeason(ctx context.Context, farm *uuid.UUID, name string, startMonth, startDay, endMonth, endDay int32) (models.Season, error) {
	row, err := r.queries.InsertSeason(ctx, database.InsertSeasonParams{
		Farm:       farm,
		Name:       name,
		StartMonth: int16(startMonth),
		StartDay:   int16(startDay),
		EndMonth:   int16(endMonth),
		EndDay:     int16(endDay),
	})
	if err != nil {
		return models.Season{}, mapForeignKeyError(err)
	}
	return toSeason(row), nil
}

func (r *seasonRepository) UpdateSeason(ctx context.Context, id uuid.UUID, name string, startMonth, startDay, endMonth, endDay int32) (models.Season, error) {
	row, err := r.queries.UpdateSeason(ctx, database.UpdateSeasonParams{
		ID:         id,
		Name:       name,
		StartMonth: int16(startMonth),
		StartDay:   int16(startDay),
		EndMonth:   int16(endMonth),
		EndDay:     int16(endDay),
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return models.Season{}, fmt.Errorf("db error: %w %w", err, services.ErrNotFound)
		}
		return models.Season{}, err
	}
	return toSeason(row), nil
}

func (r *seasonRepository) DeleteSeason(ctx context.Context, id uuid.UUID) error {
	deleted, err := r.queries.DeleteSeason(ctx, id)
	if err != nil {
		return err
	}
	if deleted == 0 {
		return services.ErrNotFound
	}
	return nil
}

func (r *seasonRepository) GetSeasonByID(ctx context.Context, id uuid.UUID) (models.Season, error) {
	row, err := r.queries.GetSeasonByID(ctx, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return models.Season{}, fmt.Errorf("db error: %w %w", err, services.ErrNotFound)
		}
		return models.Season{}, err
	}
	return toSeason(row), nil
}

func (r *seasonRepository) GetDefaultSeasons(ctx context.Context) ([]models.Season, error) {
	rows, err := r.queries.GetDefaultSeasons(ctx)
	if err != nil {
		return nil, err
	}
	seasons := make([]models.Season, len(rows))
	for i, row := range rows {
		seasons[i] = toSeason(row)
	}
	return seasons, nil
}

func (r *seasonRepository) GetFarmSeasons(ctx context.Context, farm uuid.UUID) ([]models.Season, error) {
	rows, err := r.queries.GetFarmSeasons(ctx, &farm)
	if err != nil {
		return nil, err
	}
	seasons := make([]models.Season, len(rows))
	for i, row := range rows {
		seasons[i] = toSeason(row)
	}
	return seasons, nil
}

func (r *seasonRepository) CreateCropSeasonRule(ctx context.Context, crop uuid.UUID, farm *uuid.UUID, season uuid.UUID) (models.CropSeasonRule, error) {
	row, err := r.queries.InsertCropSeasonRule(ctx, database.InsertCropSeasonRuleParams{
		SeasonID: season,
		CropID:   crop,
		FarmID:   farm,
	})
	if err != nil {
		if isUniqueViolation(err) {
			return models.CropSeasonRule{}, services.ErrCropSeasonRuleExists
		}
		return models.CropSeasonRule{}, mapForeignKeyError(err)
	}
	return toCropSeasonRule(row), nil
}

func (r *seasonRepository) UpdateCropSeasonRule(ctx context.Context, id uuid.UUID, season uuid.UUID) (models.CropSeasonRule, error) {
	row, err := r.queries.UpdateCropSeasonRule(ctx, database.UpdateCropSeasonRuleParams{
		ID:       id,
		SeasonID: season,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return models.CropSeasonRule{}, fmt.Errorf("db error: %w %w", err, services.ErrNotFound)
		}
		return models.CropSeasonRule{}, mapForeignKeyError(err)
	}
	return toCropSeasonRule(row), nil
}

func (r *seasonRepository) DeleteCropSeasonRule(ctx context.Context, id uuid.UUID) error {
	deleted, err := r.queries.DeleteCropSeasonRule(ctx, id)
	if err != nil {
		return err
	}
	if deleted == 0 {
		return services.ErrNotFound
	}
	return nil
}

func (r *seasonRepository) GetCropSeasonRuleByID(ctx context.Context, id uuid.UUID) (models.CropSeasonRule, error) {
	row, err := r.queries.GetCropSeasonRuleByID(ctx, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return models.CropSeasonRule{}, fmt.Errorf("db error: %w %w", err, services.ErrNotFound)
		}
		return models.CropSeasonRule{}, err
	}
	return toCropSeasonRule(row), nil
}

func (r *seasonRepository) GetCropSeasonRuleForCrop(ctx context.Context, crop uuid.UUID, farm *uuid.UUID) (models.CropSeasonRule, error) {
	row, err := r.queries.GetCropSeasonRuleForCrop(ctx, database.GetCropSeasonRuleForCropParams{Crop: crop, Farm: farm})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return models.CropSeasonRule{}, fmt.Errorf("db error: %w %w", err, services.ErrNotFound)
		}
		return models.CropSeasonRule{}, err
	}
	return toCropSeasonRule(row), nil
}

// GetEffectiveSeasonForCrop returns the season a crop is checked against for
// a given farm, and whether it is restricted at all: false means the crop
// has no rule, default or farm-owned, and can be rented year-round.
func (r *seasonRepository) GetEffectiveSeasonForCrop(ctx context.Context, crop, farm uuid.UUID) (models.Season, bool, error) {
	row, err := r.queries.GetEffectiveSeasonForCrop(ctx, database.GetEffectiveSeasonForCropParams{Crop: crop, Farm: &farm})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return models.Season{}, false, nil
		}
		return models.Season{}, false, err
	}
	return toSeason(row), true, nil
}

// GetEffectiveSeasonsForCrops is GetEffectiveSeasonForCrop batched over
// several (crop, farm) pairs in one round trip. A pair absent from the
// returned map has no rule, default or farm-owned, and is unrestricted.
func (r *seasonRepository) GetEffectiveSeasonsForCrops(ctx context.Context, pairs []models.CropAtFarm) (map[models.CropAtFarm]models.Season, error) {
	crops := make([]uuid.UUID, len(pairs))
	farms := make([]uuid.UUID, len(pairs))
	for i, pair := range pairs {
		crops[i] = pair.Crop
		farms[i] = pair.Farm
	}

	rows, err := r.queries.GetEffectiveSeasonsForCrops(ctx, database.GetEffectiveSeasonsForCropsParams{Crops: crops, Farms: farms})
	if err != nil {
		return nil, err
	}

	seasons := make(map[models.CropAtFarm]models.Season, len(rows))
	for _, row := range rows {
		key := models.CropAtFarm{Crop: row.ForCrop, Farm: row.ForFarm}
		seasons[key] = toSeason(database.Season{
			ID:         row.ID,
			Farm:       row.Farm,
			Name:       row.Name,
			StartMonth: row.StartMonth,
			StartDay:   row.StartDay,
			EndMonth:   row.EndMonth,
			EndDay:     row.EndDay,
			CreatedAt:  row.CreatedAt,
			UpdatedAt:  row.UpdatedAt,
		})
	}
	return seasons, nil
}

func toSeason(row database.Season) models.Season {
	return models.Season{
		ID:         row.ID,
		Farm:       row.Farm,
		Name:       row.Name,
		StartMonth: int32(row.StartMonth),
		StartDay:   int32(row.StartDay),
		EndMonth:   int32(row.EndMonth),
		EndDay:     int32(row.EndDay),
		CreatedAt:  row.CreatedAt.Time,
		UpdatedAt:  row.UpdatedAt.Time,
	}
}

func toCropSeasonRule(row database.SeasonCrop) models.CropSeasonRule {
	return models.CropSeasonRule{
		ID:     row.ID,
		Crop:   row.CropID,
		Season: row.SeasonID,
		Farm:   row.FarmID,
	}
}
