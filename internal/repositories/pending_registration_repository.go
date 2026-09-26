package repositories

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/Neue-Konzepte-BaaS/backend/internal/models"
	database "github.com/Neue-Konzepte-BaaS/backend/internal/repositories/db"
	"github.com/Neue-Konzepte-BaaS/backend/internal/services"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

type pendingRegistrationRepository struct {
	queries *database.Queries
}

func NewPendingRegistrationRepository(queries *database.Queries) services.PendingRegistrationRepository {
	return &pendingRegistrationRepository{queries: queries}
}

func (r *pendingRegistrationRepository) UpsertPendingRegistration(ctx context.Context, reg models.PendingRegistration, ttl time.Duration) (uuid.UUID, error) {
	id, err := r.queries.UpsertPendingRegistration(ctx, database.UpsertPendingRegistrationParams{
		FirstName:    reg.FirstName,
		LastName:     reg.LastName,
		Email:        reg.Email,
		PasswordHash: reg.PasswordHash,
		Role:         string(reg.Role),
		FarmName:     reg.FarmName,
		Address:      reg.Address,
		Description:  reg.Description,
		PostalCode:   reg.PostalCode,
		ExpiresAt:    pgtype.Timestamptz{Time: time.Now().Add(ttl), Valid: true},
	})
	if err != nil {
		return uuid.UUID{}, fmt.Errorf("upserting pending registration: %w", err)
	}
	return id, nil
}

func (r *pendingRegistrationRepository) GetPendingRegistrationByID(ctx context.Context, id uuid.UUID) (models.PendingRegistration, error) {
	row, err := r.queries.GetPendingRegistrationByID(ctx, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return models.PendingRegistration{}, fmt.Errorf("db error: %w %w", err, services.ErrNotFound)
		}
		return models.PendingRegistration{}, err
	}

	return models.PendingRegistration{
		ID:           row.ID,
		FirstName:    row.FirstName,
		LastName:     row.LastName,
		Email:        row.Email,
		PasswordHash: row.PasswordHash,
		Role:         models.Role(row.Role),
		FarmName:     row.FarmName,
		Address:      row.Address,
		Description:  row.Description,
		PostalCode:   row.PostalCode,
	}, nil
}

func (r *pendingRegistrationRepository) DeletePendingRegistration(ctx context.Context, id uuid.UUID) error {
	return r.queries.DeletePendingRegistration(ctx, id)
}
