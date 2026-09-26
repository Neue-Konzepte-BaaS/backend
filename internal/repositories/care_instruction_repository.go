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
)

type careInstructionRepository struct {
	queries *database.Queries
}

func NewCareInstructionRepository(queries *database.Queries) services.CareInstructionRepository {
	return &careInstructionRepository{queries: queries}
}

func (r *careInstructionRepository) CreateCareInstruction(ctx context.Context, crop uuid.UUID, week int32, title, body string) (models.CareInstruction, error) {
	row, err := r.queries.InsertCareInstruction(ctx, database.InsertCareInstructionParams{
		Crop:  crop,
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
	return toCareInstruction(row), nil
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

func (r *careInstructionRepository) GetCareInstructionsByCrop(ctx context.Context, crop uuid.UUID) ([]models.CareInstruction, error) {
	rows, err := r.queries.GetCareInstructionsByCrop(ctx, crop)
	if err != nil {
		return nil, err
	}

	instructions := make([]models.CareInstruction, len(rows))
	for i, row := range rows {
		instructions[i] = toCareInstruction(row)
	}
	return instructions, nil
}

func (r *careInstructionRepository) GetCareInstructionsByCrops(ctx context.Context, crops []uuid.UUID) (map[uuid.UUID][]models.CareInstruction, error) {
	rows, err := r.queries.GetCareInstructionsByCrops(ctx, crops)
	if err != nil {
		return nil, err
	}

	// The query already orders by week, so appending in row order keeps each
	// crop's guide sorted without a second sort per crop.
	byCrop := make(map[uuid.UUID][]models.CareInstruction, len(crops))
	for _, row := range rows {
		byCrop[row.Crop] = append(byCrop[row.Crop], toCareInstruction(row))
	}
	return byCrop, nil
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

func toCareInstruction(row database.CareInstruction) models.CareInstruction {
	return models.CareInstruction{
		ID:        row.ID,
		Crop:      row.Crop,
		Week:      row.Week,
		Title:     row.Title,
		Body:      row.Body,
		CreatedAt: row.CreatedAt.Time,
		UpdatedAt: row.UpdatedAt.Time,
	}
}
