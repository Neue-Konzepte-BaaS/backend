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

type careInstructionRepository struct {
	pool    *pgxpool.Pool
	queries *database.Queries
}

func NewCareInstructionRepository(pool *pgxpool.Pool, queries *database.Queries) services.CareInstructionRepository {
	return &careInstructionRepository{pool: pool, queries: queries}
}

// careInstructionRow is the column set every care_instruction query returns.
// sqlc generates one row type per query; they share these fields exactly, so
// each converts to this one and a single mapper serves them all.
type careInstructionRow = database.InsertCareInstructionRow

func (r *careInstructionRepository) CreateCareInstruction(ctx context.Context, crop uuid.UUID, farm *uuid.UUID, week int32, title, body string) (models.CareInstruction, error) {
	row, err := r.queries.InsertCareInstruction(ctx, database.InsertCareInstructionParams{
		Crop:  crop,
		Farm:  farm,
		Week:  week,
		Title: title,
		Body:  body,
	})
	if err != nil {
		return models.CareInstruction{}, mapCareInstructionError(err)
	}
	return toCareInstruction(row), nil
}

func (r *careInstructionRepository) UpdateCareInstruction(ctx context.Context, id uuid.UUID, week int32, title, body string) (models.CareInstruction, error) {
	row, err := r.queries.UpdateCareInstruction(ctx, database.UpdateCareInstructionParams{
		ID:    id,
		Week:  week,
		Title: title,
		Body:  body,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return models.CareInstruction{}, fmt.Errorf("db error: %w %w", err, services.ErrNotFound)
		}
		return models.CareInstruction{}, mapCareInstructionError(err)
	}
	return toCareInstruction(careInstructionRow(row)), nil
}

func (r *careInstructionRepository) DeleteCareInstruction(ctx context.Context, id uuid.UUID) error {
	deleted, err := r.queries.DeleteCareInstruction(ctx, id)
	if err != nil {
		return err
	}
	if deleted == 0 {
		return services.ErrNotFound
	}
	return nil
}

func (r *careInstructionRepository) GetCareInstructionByID(ctx context.Context, id uuid.UUID) (models.CareInstruction, error) {
	row, err := r.queries.GetCareInstructionByID(ctx, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return models.CareInstruction{}, fmt.Errorf("db error: %w %w", err, services.ErrNotFound)
		}
		return models.CareInstruction{}, err
	}
	return toCareInstruction(careInstructionRow(row)), nil
}

func (r *careInstructionRepository) GetDefaultCareInstructionsByCrop(ctx context.Context, crop uuid.UUID) ([]models.CareInstruction, error) {
	rows, err := r.queries.GetDefaultCareInstructionsByCrop(ctx, crop)
	if err != nil {
		return nil, err
	}

	instructions := make([]models.CareInstruction, len(rows))
	for i, row := range rows {
		instructions[i] = toCareInstruction(careInstructionRow(row))
	}
	return instructions, nil
}

// StartFarmCareGuide inserts the marker and copies the default in one
// transaction, so a failed copy never leaves a farm with an empty guide it did
// not ask for. The marker's primary key serializes two first edits at once:
// the second waits for the first to commit, then inserts nothing and so
// copies nothing.
func (r *careInstructionRepository) StartFarmCareGuide(ctx context.Context, crop, farm uuid.UUID) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }() // no-op after a successful Commit

	qtx := r.queries.WithTx(tx)

	started, err := qtx.InsertFarmCareGuide(ctx, database.InsertFarmCareGuideParams{Farm: farm, Crop: crop})
	if err != nil {
		return mapForeignKeyError(err)
	}
	if started == 0 {
		return nil
	}

	if err := qtx.CopyDefaultCareInstructionsToFarm(ctx, database.CopyDefaultCareInstructionsToFarmParams{Farm: farm, Crop: crop}); err != nil {
		return fmt.Errorf("copying default care guide: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit tx: %w", err)
	}
	return nil
}

func (r *careInstructionRepository) GetFarmCopyOfCareInstruction(ctx context.Context, farm, basedOn uuid.UUID) (models.CareInstruction, error) {
	row, err := r.queries.GetFarmCopyOfCareInstruction(ctx, database.GetFarmCopyOfCareInstructionParams{Farm: &farm, BasedOn: &basedOn})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return models.CareInstruction{}, fmt.Errorf("db error: %w %w", err, services.ErrNotFound)
		}
		return models.CareInstruction{}, err
	}
	return toCareInstruction(careInstructionRow(row)), nil
}

func (r *careInstructionRepository) DeleteFarmCareGuide(ctx context.Context, crop, farm uuid.UUID) error {
	deleted, err := r.queries.DeleteFarmCareGuide(ctx, database.DeleteFarmCareGuideParams{Farm: farm, Crop: crop})
	if err != nil {
		return err
	}
	if deleted == 0 {
		return services.ErrNotFound
	}
	return nil
}

func (r *careInstructionRepository) HasFarmCareGuide(ctx context.Context, crop, farm uuid.UUID) (bool, error) {
	return r.queries.HasFarmCareGuide(ctx, database.HasFarmCareGuideParams{Farm: farm, Crop: crop})
}

func (r *careInstructionRepository) GetEffectiveCareInstructions(ctx context.Context, guides []models.CropAtFarm) (map[models.CropAtFarm][]models.CareInstruction, error) {
	crops := make([]uuid.UUID, len(guides))
	farms := make([]uuid.UUID, len(guides))
	for i, guide := range guides {
		crops[i] = guide.Crop
		farms[i] = guide.Farm
	}

	rows, err := r.queries.GetEffectiveCareInstructions(ctx, database.GetEffectiveCareInstructionsParams{Crops: crops, Farms: farms})
	if err != nil {
		return nil, err
	}

	// The query already orders by week, so appending in row order keeps each
	// guide sorted without a second sort per guide.
	byGuide := make(map[models.CropAtFarm][]models.CareInstruction, len(guides))
	for _, row := range rows {
		key := models.CropAtFarm{Crop: row.ForCrop, Farm: row.ForFarm}
		byGuide[key] = append(byGuide[key], toCareInstruction(careInstructionRow{
			ID:        row.ID,
			Crop:      row.Crop,
			Farm:      row.Farm,
			Week:      row.Week,
			Title:     row.Title,
			Body:      row.Body,
			CreatedAt: row.CreatedAt,
			UpdatedAt: row.UpdatedAt,
		}))
	}
	return byGuide, nil
}

// mapCareInstructionError adds the one violation the shared foreign-key
// mapping does not cover: a week outside the table's CHECK, which is a 400
// rather than a 404. An unknown crop id is the ordinary foreign-key case, so
// it falls through to mapForeignKeyError. Both are rejected in the handler
// first; the mapping exists so a request that slips past it — a crop deleted
// between validation and insert — still reads as what it is.
func mapCareInstructionError(err error) error {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == checkViolation {
		return fmt.Errorf("%s: %w", pgErr.Message, services.ErrInvalidCareInstruction)
	}
	return mapForeignKeyError(err)
}

func toCareInstruction(row careInstructionRow) models.CareInstruction {
	return models.CareInstruction{
		ID:        row.ID,
		Crop:      row.Crop,
		Farm:      row.Farm,
		Week:      row.Week,
		Title:     row.Title,
		Body:      row.Body,
		CreatedAt: row.CreatedAt.Time,
		UpdatedAt: row.UpdatedAt.Time,
	}
}
