package models

import (
	"time"

	"github.com/google/uuid"
)

// CheckoutStatus is the lifecycle state of a Stripe Checkout Session as
// tracked locally, from creation until a rental exists (or the payment
// fails to produce one).
type CheckoutStatus string

const (
	CheckoutStatusPending   CheckoutStatus = "pending"
	CheckoutStatusCompleted CheckoutStatus = "completed"
	CheckoutStatusFailed    CheckoutStatus = "failed"
	CheckoutStatusExpired   CheckoutStatus = "expired"
	CheckoutStatusRefunded  CheckoutStatus = "refunded"
)

// RentalCheckout tracks one Stripe Checkout Session from creation until a
// rental request exists from it, or the payment fails to produce one. It
// carries the rental request's own inputs (StartAt, Message) so the Stripe
// webhook -- which only ever supplies a session id -- can call
// RentalService.RequestRental without asking the customer again.
type RentalCheckout struct {
	ID                      uuid.UUID
	Customer                uuid.UUID
	Plot                    uuid.UUID
	Crop                    uuid.UUID
	StartAt                 time.Time
	Message                 string
	StripeCheckoutSessionID string
	Status                  CheckoutStatus
	AmountCents             int32
	// Rental is set once Status is Completed or Refunded, nil otherwise.
	Rental    *uuid.UUID
	CreatedAt time.Time
	UpdatedAt time.Time
}

// CheckoutSessionResult is what creating a checkout session hands back to
// the customer: enough to render the embedded payment iframe and the
// summary above it in one round trip.
type CheckoutSessionResult struct {
	ClientSecret string
	PlotName     string
	CropName     string
	PriceCents   int32
}

// CheckoutSessionStatusResult is what polling a checkout session's status
// returns. Rental is set once the payment produced one, whether it is
// still Requested, Approved, or Declined.
type CheckoutSessionStatusResult struct {
	Status CheckoutStatus
	Rental *Rental
}
