package models

import (
	"time"

	"github.com/google/uuid"
)

// FarmerSubscriptionStatus is the lifecycle state of a farmer's subscription,
// tracked locally and kept in sync with Stripe via webhook.
type FarmerSubscriptionStatus string

const (
	FarmerSubscriptionPending  FarmerSubscriptionStatus = "pending"
	FarmerSubscriptionActive   FarmerSubscriptionStatus = "active"
	FarmerSubscriptionPastDue  FarmerSubscriptionStatus = "past_due"
	FarmerSubscriptionCanceled FarmerSubscriptionStatus = "canceled"
)

// FarmerSubscription tracks one farmer's subscription lifecycle from
// checkout creation onward. It is the payment ledger; Stripe's own
// subscription object remains the source of truth for billing itself.
type FarmerSubscription struct {
	ID                      uuid.UUID
	Farmer                  uuid.UUID
	Plan                    uuid.UUID
	StripeCustomerID        string
	StripeSubscriptionID    *string
	StripeCheckoutSessionID *string
	Status                  FarmerSubscriptionStatus
	CurrentPeriodEnd        *time.Time
	CreatedAt               time.Time
	UpdatedAt               time.Time
}

// SubscriptionCheckoutResult is what opening a subscription checkout hands
// back to the farmer, mirroring CheckoutSessionResult for the rental flow.
type SubscriptionCheckoutResult struct {
	ClientSecret string
	PlanCode     SubscriptionPlanCode
	PriceCents   int32
}
