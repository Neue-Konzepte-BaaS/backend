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

type accountRepository struct {
	queries *database.Queries
}

func NewAccountRepository(queries *database.Queries) services.AccountRepository {
	return &accountRepository{queries: queries}
}

func (r *accountRepository) GetAccountByEmail(ctx context.Context, email string) (models.Account, error) {
	row, err := r.queries.GetAccountByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return models.Account{}, fmt.Errorf("db error: %w %w", err, services.ErrNotFound)
		} else {
			return models.Account{}, err
		}
	}

	return models.Account{
		ID:           row.ID,
		FirstName:    row.FirstName,
		LastName:     row.LastName,
		Email:        row.Email,
		PasswordHash: row.PasswordHash,
		Role:         models.Role(row.Role),
	}, nil
}

func (r *accountRepository) GetAccountByID(ctx context.Context, id uuid.UUID) (models.Account, error) {
	row, err := r.queries.GetAccountByID(ctx, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return models.Account{}, fmt.Errorf("db error: %w %w", err, services.ErrNotFound)
		} else {
			return models.Account{}, err
		}
	}

	return models.Account{
		ID:        row.ID,
		FirstName: row.FirstName,
		LastName:  row.LastName,
		Email:     row.Email,
		Role:      models.Role(row.Role),
	}, nil
}
