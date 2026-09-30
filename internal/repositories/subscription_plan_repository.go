package repositories

import (
	"context"
	"errors"

	"github.com/Neue-Konzepte-BaaS/backend/internal/models"
	database "github.com/Neue-Konzepte-BaaS/backend/internal/repositories/db"
	"github.com/Neue-Konzepte-BaaS/backend/internal/services"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

type subscriptionPlanRepository struct {
	queries *database.Queries
}

func NewSubscriptionPlanRepository(queries *database.Queries) services.SubscriptionPlanRepository {
	return &subscriptionPlanRepository{queries: queries}
}

func (r *subscriptionPlanRepository) CreateSubscriptionPlan(ctx context.Context, code models.SubscriptionPlanCode, displayName string, maxPlots *int32, priceCents int32, stripePriceID string) error {
	var maxPlotsParam pgtype.Int4
	if maxPlots != nil {
		maxPlotsParam = pgtype.Int4{Int32: *maxPlots, Valid: true}
	}
	return r.queries.InsertSubscriptionPlan(ctx, database.InsertSubscriptionPlanParams{
		Code:          string(code),
		DisplayName:   displayName,
		MaxPlots:      maxPlotsParam,
		PriceCents:    priceCents,
		StripePriceID: stripePriceID,
	})
}

func (r *subscriptionPlanRepository) GetActiveSubscriptionPlans(ctx context.Context) ([]models.SubscriptionPlan, error) {
	rows, err := r.queries.GetActiveSubscriptionPlans(ctx)
	if err != nil {
		return nil, err
	}
	plans := make([]models.SubscriptionPlan, len(rows))
	for i, row := range rows {
		plans[i] = toModelSubscriptionPlan(row)
	}
	return plans, nil
}

func (r *subscriptionPlanRepository) GetSubscriptionPlanByID(ctx context.Context, id uuid.UUID) (models.SubscriptionPlan, error) {
	row, err := r.queries.GetSubscriptionPlanByID(ctx, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return models.SubscriptionPlan{}, services.ErrNotFound
		}
		return models.SubscriptionPlan{}, err
	}
	return toModelSubscriptionPlan(row), nil
}

func (r *subscriptionPlanRepository) GetSubscriptionPlanByCode(ctx context.Context, code models.SubscriptionPlanCode) (models.SubscriptionPlan, error) {
	row, err := r.queries.GetSubscriptionPlanByCode(ctx, string(code))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return models.SubscriptionPlan{}, services.ErrNotFound
		}
		return models.SubscriptionPlan{}, err
	}
	return toModelSubscriptionPlan(row), nil
}

func (r *subscriptionPlanRepository) ListSubscriptionPlans(ctx context.Context) ([]models.SubscriptionPlan, error) {
	rows, err := r.queries.ListSubscriptionPlans(ctx)
	if err != nil {
		return nil, err
	}
	plans := make([]models.SubscriptionPlan, len(rows))
	for i, row := range rows {
		plans[i] = toModelSubscriptionPlan(row)
	}
	return plans, nil
}

func (r *subscriptionPlanRepository) UpdateSubscriptionPlanPrice(ctx context.Context, id uuid.UUID, priceCents int32, stripePriceID string) (models.SubscriptionPlan, error) {
	row, err := r.queries.UpdateSubscriptionPlanPrice(ctx, database.UpdateSubscriptionPlanPriceParams{
		ID:            id,
		PriceCents:    priceCents,
		StripePriceID: stripePriceID,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return models.SubscriptionPlan{}, services.ErrNotFound
		}
		return models.SubscriptionPlan{}, err
	}
	return toModelSubscriptionPlan(row), nil
}

func (r *subscriptionPlanRepository) SetSubscriptionPlanActive(ctx context.Context, id uuid.UUID, active bool) error {
	return r.queries.SetSubscriptionPlanActive(ctx, database.SetSubscriptionPlanActiveParams{ID: id, IsActive: active})
}

func toModelSubscriptionPlan(row database.SubscriptionPlan) models.SubscriptionPlan {
	var maxPlots *int32
	if row.MaxPlots.Valid {
		v := row.MaxPlots.Int32
		maxPlots = &v
	}
	return models.SubscriptionPlan{
		ID:            row.ID,
		Code:          models.SubscriptionPlanCode(row.Code),
		DisplayName:   row.DisplayName,
		MaxPlots:      maxPlots,
		PriceCents:    row.PriceCents,
		StripePriceID: row.StripePriceID,
		IsActive:      row.IsActive,
		CreatedAt:     row.CreatedAt.Time,
		UpdatedAt:     row.UpdatedAt.Time,
	}
}
