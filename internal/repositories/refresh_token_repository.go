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

type refreshTokenRepository struct {
	queries *database.Queries
}

func NewRefreshTokenRepository(queries *database.Queries) services.RefreshTokenRepository {
	return &refreshTokenRepository{queries: queries}
}

func (r *refreshTokenRepository) InsertRefreshToken(ctx context.Context, accountID uuid.UUID, tokenHash string, expiresAt time.Time) error {
	if err := r.queries.InsertRefreshToken(ctx, database.InsertRefreshTokenParams{
		AccountID: accountID,
		TokenHash: tokenHash,
		ExpiresAt: pgtype.Timestamptz{Time: expiresAt, Valid: true},
	}); err != nil {
		return fmt.Errorf("inserting refresh token: %w", err)
	}
	return nil
}

func (r *refreshTokenRepository) GetActiveRefreshTokenByHash(ctx context.Context, tokenHash string) (models.RefreshToken, error) {
	row, err := r.queries.GetActiveRefreshTokenByHash(ctx, tokenHash)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return models.RefreshToken{}, fmt.Errorf("db error: %w %w", err, services.ErrNotFound)
		}
		return models.RefreshToken{}, err
	}

	return models.RefreshToken{
		ID:        row.ID,
		AccountID: row.AccountID,
		TokenHash: row.TokenHash,
		ExpiresAt: row.ExpiresAt.Time,
		CreatedAt: row.CreatedAt.Time,
	}, nil
}

func (r *refreshTokenRepository) RevokeRefreshTokenByHash(ctx context.Context, tokenHash string) error {
	return r.queries.RevokeRefreshTokenByHash(ctx, tokenHash)
}
