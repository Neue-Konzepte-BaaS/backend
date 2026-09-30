package models

import (
	"time"

	"github.com/google/uuid"
)

// SubscriptionPlanCode identifies one of the fixed subscription tiers. It is
// a closed catalog (enforced by a CHECK constraint), not a farmer- or
// admin-editable label.
type SubscriptionPlanCode string

const (
	SubscriptionPlanCheap     SubscriptionPlanCode = "cheap"
	SubscriptionPlanModest    SubscriptionPlanCode = "modest"
	SubscriptionPlanExpensive SubscriptionPlanCode = "expensive"
)

// SubscriptionPlan is one of the three tiers a farmer may subscribe to.
// Prices are editable by an admin for future subscriptions only: Stripe
// Prices are immutable, so editing a plan repoints StripePriceID at a newly
// created one rather than mutating it (see the price_cents/stripe_price_id
// columns' migration comment).
type SubscriptionPlan struct {
	ID          uuid.UUID
	Code        SubscriptionPlanCode
	DisplayName string
	// MaxPlots is the most plots a farm on this plan may simultaneously
	// offer. nil means unlimited.
	MaxPlots      *int32
	PriceCents    int32
	StripePriceID string
	IsActive      bool
	CreatedAt     time.Time
	UpdatedAt     time.Time
}
