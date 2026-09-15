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

type farmRepository struct {
	queries *database.Queries
}

func NewFarmRepository(queries *database.Queries) services.FarmRepository {
	return &farmRepository{queries: queries}
}

func (r *farmRepository) GetFarmByID(ctx context.Context, farmID uuid.UUID) (models.Farm, error) {
	row, err := r.queries.GetFarmByID(ctx, farmID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return models.Farm{}, fmt.Errorf("db error: %w %w", err, services.ErrNotFound)
		}
		return models.Farm{}, err
	}

	farm := models.Farm{
		ID:                row.ID,
		FarmerID:          row.FarmerID,
		Name:              row.Name,
		Address:           row.Address,
		Description:       row.Description,
		TotalSquareMeters: row.TotalSquareMeters,
	}
	if row.FoundedAt.Valid {
		founded := row.FoundedAt.Time
		farm.FoundedAt = &founded
	}
	return farm, nil
}

// GetFarmIDByFarmerID returns ErrNotFound if the account is not a farmer.
func (r *farmRepository) GetFarmIDByFarmerID(ctx context.Context, farmerID uuid.UUID) (uuid.UUID, error) {
	id, err := r.queries.GetFarmIDByFarmerID(ctx, farmerID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return uuid.UUID{}, fmt.Errorf("db error: %w %w", err, services.ErrNotFound)
		}
		return uuid.UUID{}, err
	}
	return id, nil
}
