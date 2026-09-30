package repositories

import (
	"context"
	"errors"
	"time"

	"github.com/Neue-Konzepte-BaaS/backend/internal/models"
	database "github.com/Neue-Konzepte-BaaS/backend/internal/repositories/db"
	"github.com/Neue-Konzepte-BaaS/backend/internal/services"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

type farmerSubscriptionRepository struct {
	queries *database.Queries
}

func NewFarmerSubscriptionRepository(queries *database.Queries) services.FarmerSubscriptionRepository {
	return &farmerSubscriptionRepository{queries: queries}
}

func (r *farmerSubscriptionRepository) CreateSubscription(ctx context.Context, sub models.FarmerSubscription) (models.FarmerSubscription, error) {
	sessionID := ""
	if sub.StripeCheckoutSessionID != nil {
		sessionID = *sub.StripeCheckoutSessionID
	}
	row, err := r.queries.InsertFarmerSubscription(ctx, database.InsertFarmerSubscriptionParams{
		Farmer:                  sub.Farmer,
		Plan:                    sub.Plan,
		StripeCustomerID:        sub.StripeCustomerID,
		StripeCheckoutSessionID: textOrNull(sessionID),
	})
	if err != nil {
		return models.FarmerSubscription{}, err
	}
	return toModelFarmerSubscription(row), nil
}

func (r *farmerSubscriptionRepository) GetSubscriptionBySessionID(ctx context.Context, sessionID string) (models.FarmerSubscription, error) {
	row, err := r.queries.GetFarmerSubscriptionBySessionID(ctx, pgtype.Text{String: sessionID, Valid: true})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return models.FarmerSubscription{}, services.ErrNotFound
		}
		return models.FarmerSubscription{}, err
	}
	return toModelFarmerSubscription(row), nil
}

func (r *farmerSubscriptionRepository) GetSubscriptionByStripeSubscriptionID(ctx context.Context, stripeSubscriptionID string) (models.FarmerSubscription, error) {
	row, err := r.queries.GetFarmerSubscriptionByStripeSubscriptionID(ctx, pgtype.Text{String: stripeSubscriptionID, Valid: true})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return models.FarmerSubscription{}, services.ErrNotFound
		}
		return models.FarmerSubscription{}, err
	}
	return toModelFarmerSubscription(row), nil
}

func (r *farmerSubscriptionRepository) GetActiveSubscriptionByFarmer(ctx context.Context, farmer uuid.UUID) (models.FarmerSubscription, error) {
	row, err := r.queries.GetActiveSubscriptionByFarmer(ctx, farmer)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return models.FarmerSubscription{}, services.ErrNotFound
		}
		return models.FarmerSubscription{}, err
	}
	return toModelFarmerSubscription(row), nil
}

func (r *farmerSubscriptionRepository) GetSubscriptionByFarmer(ctx context.Context, farmer uuid.UUID) (models.FarmerSubscription, error) {
	row, err := r.queries.GetSubscriptionByFarmer(ctx, farmer)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return models.FarmerSubscription{}, services.ErrNotFound
		}
		return models.FarmerSubscription{}, err
	}
	return toModelFarmerSubscription(row), nil
}

func (r *farmerSubscriptionRepository) ActivateSubscription(ctx context.Context, id uuid.UUID, stripeSubscriptionID string, currentPeriodEnd time.Time) (models.FarmerSubscription, error) {
	row, err := r.queries.ActivateFarmerSubscription(ctx, database.ActivateFarmerSubscriptionParams{
		ID:                   id,
		StripeSubscriptionID: pgtype.Text{String: stripeSubscriptionID, Valid: true},
		CurrentPeriodEnd:     pgtype.Timestamptz{Time: currentPeriodEnd, Valid: true},
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return models.FarmerSubscription{}, services.ErrCheckoutAlreadyProcessed
		}
		return models.FarmerSubscription{}, err
	}
	return toModelFarmerSubscription(row), nil
}

func (r *farmerSubscriptionRepository) ExpireSubscription(ctx context.Context, id uuid.UUID) (models.FarmerSubscription, error) {
	row, err := r.queries.ExpireFarmerSubscription(ctx, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return models.FarmerSubscription{}, services.ErrCheckoutAlreadyProcessed
		}
		return models.FarmerSubscription{}, err
	}
	return toModelFarmerSubscription(row), nil
}

func (r *farmerSubscriptionRepository) MarkSubscriptionPastDue(ctx context.Context, stripeSubscriptionID string) (models.FarmerSubscription, error) {
	row, err := r.queries.MarkFarmerSubscriptionPastDue(ctx, pgtype.Text{String: stripeSubscriptionID, Valid: true})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return models.FarmerSubscription{}, services.ErrCheckoutAlreadyProcessed
		}
		return models.FarmerSubscription{}, err
	}
	return toModelFarmerSubscription(row), nil
}

func (r *farmerSubscriptionRepository) ReactivateSubscription(ctx context.Context, stripeSubscriptionID string, currentPeriodEnd time.Time) (models.FarmerSubscription, error) {
	row, err := r.queries.ReactivateFarmerSubscription(ctx, database.ReactivateFarmerSubscriptionParams{
		StripeSubscriptionID: pgtype.Text{String: stripeSubscriptionID, Valid: true},
		CurrentPeriodEnd:     pgtype.Timestamptz{Time: currentPeriodEnd, Valid: true},
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return models.FarmerSubscription{}, services.ErrCheckoutAlreadyProcessed
		}
		return models.FarmerSubscription{}, err
	}
	return toModelFarmerSubscription(row), nil
}

func (r *farmerSubscriptionRepository) CancelSubscription(ctx context.Context, stripeSubscriptionID string) (models.FarmerSubscription, error) {
	row, err := r.queries.CancelFarmerSubscription(ctx, pgtype.Text{String: stripeSubscriptionID, Valid: true})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return models.FarmerSubscription{}, services.ErrCheckoutAlreadyProcessed
		}
		return models.FarmerSubscription{}, err
	}
	return toModelFarmerSubscription(row), nil
}

func textOrNull(s string) pgtype.Text {
	return pgtype.Text{String: s, Valid: s != ""}
}

func toModelFarmerSubscription(row database.FarmerSubscription) models.FarmerSubscription {
	var stripeSubscriptionID *string
	if row.StripeSubscriptionID.Valid {
		v := row.StripeSubscriptionID.String
		stripeSubscriptionID = &v
	}
	var stripeCheckoutSessionID *string
	if row.StripeCheckoutSessionID.Valid {
		v := row.StripeCheckoutSessionID.String
		stripeCheckoutSessionID = &v
	}
	var currentPeriodEnd *time.Time
	if row.CurrentPeriodEnd.Valid {
		v := row.CurrentPeriodEnd.Time
		currentPeriodEnd = &v
	}
	return models.FarmerSubscription{
		ID:                      row.ID,
		Farmer:                  row.Farmer,
		Plan:                    row.Plan,
		StripeCustomerID:        row.StripeCustomerID,
		StripeSubscriptionID:    stripeSubscriptionID,
		StripeCheckoutSessionID: stripeCheckoutSessionID,
		Status:                  models.FarmerSubscriptionStatus(row.Status),
		CurrentPeriodEnd:        currentPeriodEnd,
		CreatedAt:               row.CreatedAt.Time,
		UpdatedAt:               row.UpdatedAt.Time,
	}
}
