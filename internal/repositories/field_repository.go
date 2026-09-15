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
)

type fieldRepository struct {
	queries *database.Queries
}

func NewFieldRepository(queries *database.Queries) services.FieldRepository {
	return &fieldRepository{queries: queries}
}

func (r *fieldRepository) CreateField(ctx context.Context, field models.Field) (uuid.UUID, error) {
	id, err := r.queries.InsertField(ctx, database.InsertFieldParams{
		Name:        field.Name,
		Farm:        field.Farm,
		Coordinates: field.Coordinates,
	})
	if err != nil {
		return uuid.UUID{}, mapGeometryError(err)
	}
	return id, nil
}

func (r *fieldRepository) GetFieldFarm(ctx context.Context, id uuid.UUID) (uuid.UUID, error) {
	farm, err := r.queries.GetFieldFarm(ctx, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return uuid.UUID{}, fmt.Errorf("db error: %w %w", err, services.ErrNotFound)
		}
		return uuid.UUID{}, err
	}
	return farm, nil
}

func (r *fieldRepository) GetFieldsByFarm(ctx context.Context, farm uuid.UUID) ([]models.Field, error) {
	rows, err := r.queries.GetFieldsByFarm(ctx, farm)
	if err != nil {
		return nil, err
	}

	fields := make([]models.Field, len(rows))
	for i, row := range rows {
		fields[i] = models.Field{
			ID:          row.ID,
			Name:        row.Name,
			Farm:        row.Farm,
			Coordinates: row.Coordinates,
		}
	}
	return fields, nil
}
