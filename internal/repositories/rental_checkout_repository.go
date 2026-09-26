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

type rentalCheckoutRepository struct {
	queries *database.Queries
}

func NewRentalCheckoutRepository(queries *database.Queries) services.RentalCheckoutRepository {
	return &rentalCheckoutRepository{queries: queries}
}

func (r *rentalCheckoutRepository) CreateCheckout(ctx context.Context, checkout models.RentalCheckout) (models.RentalCheckout, error) {
	row, err := r.queries.InsertRentalCheckout(ctx, database.InsertRentalCheckoutParams{
		Customer:                checkout.Customer,
		Plot:                    checkout.Plot,
		Crop:                    checkout.Crop,
		StartAt:                 pgtype.Timestamptz{Time: checkout.StartAt, Valid: true},
		Message:                 checkout.Message,
		StripeCheckoutSessionID: checkout.StripeCheckoutSessionID,
		AmountCents:             checkout.AmountCents,
	})
	if err != nil {
		return models.RentalCheckout{}, err
	}
	return toModelRentalCheckout(row.ID, row.Customer, row.Plot, row.Crop, row.StartAt, row.Message, row.StripeCheckoutSessionID, row.Status, row.AmountCents, row.Rental, row.CreatedAt, row.UpdatedAt), nil
}

func (r *rentalCheckoutRepository) GetCheckoutBySessionID(ctx context.Context, sessionID string) (models.RentalCheckout, error) {
	row, err := r.queries.GetRentalCheckoutBySessionID(ctx, sessionID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return models.RentalCheckout{}, services.ErrNotFound
		}
		return models.RentalCheckout{}, err
	}
	return toModelRentalCheckout(row.ID, row.Customer, row.Plot, row.Crop, row.StartAt, row.Message, row.StripeCheckoutSessionID, row.Status, row.AmountCents, row.Rental, row.CreatedAt, row.UpdatedAt), nil
}

func (r *rentalCheckoutRepository) CompleteCheckout(ctx context.Context, id, rental uuid.UUID) (models.RentalCheckout, error) {
	row, err := r.queries.CompleteRentalCheckout(ctx, database.CompleteRentalCheckoutParams{ID: id, Rental: rental})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return models.RentalCheckout{}, services.ErrCheckoutAlreadyProcessed
		}
		return models.RentalCheckout{}, err
	}
	return toModelRentalCheckout(row.ID, row.Customer, row.Plot, row.Crop, row.StartAt, row.Message, row.StripeCheckoutSessionID, row.Status, row.AmountCents, row.Rental, row.CreatedAt, row.UpdatedAt), nil
}

func (r *rentalCheckoutRepository) FailCheckout(ctx context.Context, id uuid.UUID) (models.RentalCheckout, error) {
	row, err := r.queries.FailRentalCheckout(ctx, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return models.RentalCheckout{}, services.ErrCheckoutAlreadyProcessed
		}
		return models.RentalCheckout{}, err
	}
	return toModelRentalCheckout(row.ID, row.Customer, row.Plot, row.Crop, row.StartAt, row.Message, row.StripeCheckoutSessionID, row.Status, row.AmountCents, row.Rental, row.CreatedAt, row.UpdatedAt), nil
}

func (r *rentalCheckoutRepository) ExpireCheckout(ctx context.Context, id uuid.UUID) (models.RentalCheckout, error) {
	row, err := r.queries.ExpireRentalCheckout(ctx, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return models.RentalCheckout{}, services.ErrCheckoutAlreadyProcessed
		}
		return models.RentalCheckout{}, err
	}
	return toModelRentalCheckout(row.ID, row.Customer, row.Plot, row.Crop, row.StartAt, row.Message, row.StripeCheckoutSessionID, row.Status, row.AmountCents, row.Rental, row.CreatedAt, row.UpdatedAt), nil
}

func (r *rentalCheckoutRepository) GetCompletedCheckoutByRental(ctx context.Context, rental uuid.UUID) (models.RentalCheckout, error) {
	row, err := r.queries.GetCompletedRentalCheckoutByRental(ctx, rental)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return models.RentalCheckout{}, services.ErrNotFound
		}
		return models.RentalCheckout{}, err
	}
	return toModelRentalCheckout(row.ID, row.Customer, row.Plot, row.Crop, row.StartAt, row.Message, row.StripeCheckoutSessionID, row.Status, row.AmountCents, row.Rental, row.CreatedAt, row.UpdatedAt), nil
}

func (r *rentalCheckoutRepository) MarkCheckoutRefunded(ctx context.Context, id uuid.UUID) (models.RentalCheckout, error) {
	row, err := r.queries.MarkRentalCheckoutRefunded(ctx, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return models.RentalCheckout{}, services.ErrCheckoutAlreadyProcessed
		}
		return models.RentalCheckout{}, err
	}
	return toModelRentalCheckout(row.ID, row.Customer, row.Plot, row.Crop, row.StartAt, row.Message, row.StripeCheckoutSessionID, row.Status, row.AmountCents, row.Rental, row.CreatedAt, row.UpdatedAt), nil
}

// toModelRentalCheckout converts the fields every rental_checkout query in
// sql/queries/rental_checkout.sql returns (they all select the same
// columns) into the domain model. rental reads as uuid.Nil when the column
// is SQL NULL -- see the uuid overrides in sqlc.yml -- which this turns
// into a nil pointer.
func toModelRentalCheckout(id, customer, plot, crop uuid.UUID, startAt pgtype.Timestamptz, message, sessionID, status string, amountCents int32, rental uuid.UUID, createdAt, updatedAt pgtype.Timestamptz) models.RentalCheckout {
	checkout := models.RentalCheckout{
		ID:                      id,
		Customer:                customer,
		Plot:                    plot,
		Crop:                    crop,
		StartAt:                 startAt.Time,
		Message:                 message,
		StripeCheckoutSessionID: sessionID,
		Status:                  models.CheckoutStatus(status),
		AmountCents:             amountCents,
		CreatedAt:               createdAt.Time,
		UpdatedAt:               updatedAt.Time,
	}
	if rental != uuid.Nil {
		r := rental
		checkout.Rental = &r
	}
	return checkout
}
