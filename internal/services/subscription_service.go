package services

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/Neue-Konzepte-BaaS/backend/internal/models"
	"github.com/google/uuid"
)

// subscriptionCheckoutReturnPath mirrors checkoutReturnPath for the rental
// flow: where Stripe redirects the top-level page once a subscription
// checkout completes inside the embedded iframe.
const subscriptionCheckoutReturnPath = "/farmer/subscribe/return?session_id={CHECKOUT_SESSION_ID}"

// SubscriptionService manages a farmer's monthly subscription: choosing a
// plan, opening its Stripe Checkout session, and reacting to its lifecycle
// via webhook. It is independent of PaymentService (the rental flow) by
// design -- the two share only the underlying PaymentGateway.
type SubscriptionService interface {
	// GetAvailablePlans returns the plans currently open to new
	// subscriptions, cheapest first.
	GetAvailablePlans(ctx context.Context) ([]models.SubscriptionPlan, error)
	// GetSubscriptionStatus returns the caller's current subscription, if
	// any (including one still Pending). Returns ErrNotFound if the farmer
	// has never started a subscription.
	GetSubscriptionStatus(ctx context.Context, farmer uuid.UUID) (models.FarmerSubscription, error)
	// HasActiveSubscription reports whether the farmer currently holds an
	// Active or PastDue subscription -- the gate RequireActiveSubscription
	// checks before letting a farmer do anything else.
	HasActiveSubscription(ctx context.Context, farmer uuid.UUID) (bool, error)
	// CreateSubscriptionCheckoutSession opens a Stripe subscription checkout
	// for the calling farmer's chosen plan. Returns
	// ErrSubscriptionAlreadyActive if the farmer already has a non-canceled
	// subscription.
	CreateSubscriptionCheckoutSession(ctx context.Context, farmer, planID uuid.UUID) (models.SubscriptionCheckoutResult, error)
	// HandleWebhookEvent verifies and processes one Stripe webhook
	// delivery. It is idempotent, same guarantee as
	// PaymentService.HandleWebhookEvent, and independently no-ops on any
	// event type it does not act on (the rental-checkout event family).
	HandleWebhookEvent(ctx context.Context, payload []byte, sigHeader string) error
	// HasCapacityForAdditionalPlot reports whether the farmer's active plan
	// allows one more plot beyond currentPlotCount. Callers only reach this
	// once RequireActiveSubscription has already confirmed a subscription
	// exists, so a missing subscription here is a programming error, not a
	// normal case to handle gracefully.
	HasCapacityForAdditionalPlot(ctx context.Context, farmer uuid.UUID, currentPlotCount int64) (bool, error)
	// ListAllPlans is the admin view: every plan, active or retired.
	ListAllPlans(ctx context.Context) ([]models.SubscriptionPlan, error)
	// UpdatePlanPrice creates a new Stripe Price for the plan and repoints
	// it there. Existing subscribers keep paying whatever Price their own
	// subscription already references.
	UpdatePlanPrice(ctx context.Context, planID uuid.UUID, priceCents int32) (models.SubscriptionPlan, error)
	// SetPlanActive retires or reactivates a tier without deleting it.
	SetPlanActive(ctx context.Context, planID uuid.UUID, active bool) error
}

type subscriptionService struct {
	subscriptionRepo SubscriptionPlanRepository
	farmerSubRepo    FarmerSubscriptionRepository
	paymentGateway   PaymentGateway
	accountRepo      AccountRepository
	frontendURL      string
}

func NewSubscriptionService(farmerSubRepo FarmerSubscriptionRepository, subscriptionRepo SubscriptionPlanRepository, paymentGateway PaymentGateway, accountRepo AccountRepository, frontendURL string) SubscriptionService {
	return &subscriptionService{
		subscriptionRepo: subscriptionRepo,
		farmerSubRepo:    farmerSubRepo,
		paymentGateway:   paymentGateway,
		accountRepo:      accountRepo,
		frontendURL:      frontendURL,
	}
}

func (s *subscriptionService) GetAvailablePlans(ctx context.Context) ([]models.SubscriptionPlan, error) {
	return s.subscriptionRepo.GetActiveSubscriptionPlans(ctx)
}

func (s *subscriptionService) GetSubscriptionStatus(ctx context.Context, farmer uuid.UUID) (models.FarmerSubscription, error) {
	return s.farmerSubRepo.GetSubscriptionByFarmer(ctx, farmer)
}

func (s *subscriptionService) HasActiveSubscription(ctx context.Context, farmer uuid.UUID) (bool, error) {
	_, err := s.farmerSubRepo.GetActiveSubscriptionByFarmer(ctx, farmer)
	if errors.Is(err, ErrNotFound) {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("looking up active subscription: %w", err)
	}
	return true, nil
}

func (s *subscriptionService) CreateSubscriptionCheckoutSession(ctx context.Context, farmer, planID uuid.UUID) (models.SubscriptionCheckoutResult, error) {
	if _, err := s.farmerSubRepo.GetSubscriptionByFarmer(ctx, farmer); err == nil {
		return models.SubscriptionCheckoutResult{}, ErrSubscriptionAlreadyActive
	} else if !errors.Is(err, ErrNotFound) {
		return models.SubscriptionCheckoutResult{}, fmt.Errorf("checking existing subscription: %w", err)
	}

	plan, err := s.subscriptionRepo.GetSubscriptionPlanByID(ctx, planID)
	if err != nil {
		return models.SubscriptionCheckoutResult{}, err
	}
	if !plan.IsActive {
		return models.SubscriptionCheckoutResult{}, ErrNotFound
	}

	account, err := s.accountRepo.GetAccountByID(ctx, farmer)
	if err != nil {
		return models.SubscriptionCheckoutResult{}, fmt.Errorf("looking up farmer account: %w", err)
	}

	returnURL := s.frontendURL + subscriptionCheckoutReturnPath
	sessionID, clientSecret, stripeCustomerID, err := s.paymentGateway.CreateSubscriptionCheckoutSession(ctx, plan.StripePriceID, account.Email, nil, returnURL)
	if err != nil {
		return models.SubscriptionCheckoutResult{}, fmt.Errorf("creating stripe subscription checkout session: %w", err)
	}

	if _, err := s.farmerSubRepo.CreateSubscription(ctx, models.FarmerSubscription{
		Farmer:                  farmer,
		Plan:                    plan.ID,
		StripeCustomerID:        stripeCustomerID,
		StripeCheckoutSessionID: &sessionID,
	}); err != nil {
		return models.SubscriptionCheckoutResult{}, fmt.Errorf("recording subscription: %w", err)
	}

	return models.SubscriptionCheckoutResult{
		ClientSecret: clientSecret,
		PlanCode:     plan.Code,
		PriceCents:   plan.PriceCents,
	}, nil
}

func (s *subscriptionService) HandleWebhookEvent(ctx context.Context, payload []byte, sigHeader string) error {
	event, ok, err := s.paymentGateway.ParseWebhookEvent(payload, sigHeader)
	if err != nil {
		return err
	}
	if !ok {
		return nil // an event type this service does not act on
	}

	switch event.Type {
	case WebhookEventSubscriptionCheckoutCompleted:
		return s.completeCheckout(ctx, event.CheckoutSessionID, event.StripeSubscriptionID)
	case WebhookEventSubscriptionCheckoutExpired:
		return s.expireCheckout(ctx, event.CheckoutSessionID)
	case WebhookEventInvoicePaymentFailed:
		if _, err := s.farmerSubRepo.MarkSubscriptionPastDue(ctx, event.StripeSubscriptionID); err != nil && !errors.Is(err, ErrCheckoutAlreadyProcessed) {
			return fmt.Errorf("marking subscription past due: %w", err)
		}
		return nil
	case WebhookEventInvoicePaymentSucceeded:
		if event.CurrentPeriodEnd == nil {
			return errors.New("invoice.payment_succeeded webhook has no current_period_end")
		}
		if _, err := s.farmerSubRepo.ReactivateSubscription(ctx, event.StripeSubscriptionID, *event.CurrentPeriodEnd); err != nil && !errors.Is(err, ErrCheckoutAlreadyProcessed) {
			return fmt.Errorf("reactivating subscription: %w", err)
		}
		return nil
	case WebhookEventSubscriptionDeleted:
		if _, err := s.farmerSubRepo.CancelSubscription(ctx, event.StripeSubscriptionID); err != nil && !errors.Is(err, ErrCheckoutAlreadyProcessed) {
			return fmt.Errorf("canceling subscription: %w", err)
		}
		return nil
	}
	return nil // an event type this service does not act on (e.g. the rental-checkout family)
}

// completeCheckout activates the farmer_subscription a confirmed
// subscription checkout is for. The checkout.session.completed payload
// carries the new Stripe Subscription's id but not yet its billing period
// -- that arrives moments later via invoice.payment_succeeded, which is
// what actually sets CurrentPeriodEnd (see ReactivateSubscription, whose
// guard also accepts an Active subscription so that follow-up event is not
// a no-op). Activating here with a zero period end is a short-lived gap
// between the two webhook deliveries, not a permanent state.
func (s *subscriptionService) completeCheckout(ctx context.Context, sessionID, stripeSubscriptionID string) error {
	if stripeSubscriptionID == "" {
		return errors.New("subscription checkout completed webhook has no subscription id")
	}

	sub, err := s.farmerSubRepo.GetSubscriptionBySessionID(ctx, sessionID)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			return nil // not a subscription checkout this service created
		}
		return fmt.Errorf("looking up subscription: %w", err)
	}
	if sub.Status != models.FarmerSubscriptionPending {
		return nil
	}

	if _, err := s.farmerSubRepo.ActivateSubscription(ctx, sub.ID, stripeSubscriptionID, time.Time{}); err != nil && !errors.Is(err, ErrCheckoutAlreadyProcessed) {
		return fmt.Errorf("activating subscription: %w", err)
	}
	return nil
}

func (s *subscriptionService) expireCheckout(ctx context.Context, sessionID string) error {
	sub, err := s.farmerSubRepo.GetSubscriptionBySessionID(ctx, sessionID)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			return nil
		}
		return fmt.Errorf("looking up subscription: %w", err)
	}
	if _, err := s.farmerSubRepo.ExpireSubscription(ctx, sub.ID); err != nil && !errors.Is(err, ErrCheckoutAlreadyProcessed) {
		return fmt.Errorf("expiring subscription: %w", err)
	}
	return nil
}

func (s *subscriptionService) HasCapacityForAdditionalPlot(ctx context.Context, farmer uuid.UUID, currentPlotCount int64) (bool, error) {
	sub, err := s.farmerSubRepo.GetActiveSubscriptionByFarmer(ctx, farmer)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			return false, nil
		}
		return false, fmt.Errorf("looking up active subscription: %w", err)
	}

	plan, err := s.subscriptionRepo.GetSubscriptionPlanByID(ctx, sub.Plan)
	if err != nil {
		return false, fmt.Errorf("looking up subscription plan: %w", err)
	}
	if plan.MaxPlots == nil {
		return true, nil // unlimited tier
	}
	return currentPlotCount < int64(*plan.MaxPlots), nil
}

func (s *subscriptionService) ListAllPlans(ctx context.Context) ([]models.SubscriptionPlan, error) {
	return s.subscriptionRepo.ListSubscriptionPlans(ctx)
}

func (s *subscriptionService) UpdatePlanPrice(ctx context.Context, planID uuid.UUID, priceCents int32) (models.SubscriptionPlan, error) {
	plan, err := s.subscriptionRepo.GetSubscriptionPlanByID(ctx, planID)
	if err != nil {
		return models.SubscriptionPlan{}, err
	}

	stripePriceID, err := s.paymentGateway.CreateSubscriptionPrice(ctx, plan.DisplayName, int64(priceCents))
	if err != nil {
		return models.SubscriptionPlan{}, fmt.Errorf("creating stripe price: %w", err)
	}

	return s.subscriptionRepo.UpdateSubscriptionPlanPrice(ctx, planID, priceCents, stripePriceID)
}

func (s *subscriptionService) SetPlanActive(ctx context.Context, planID uuid.UUID, active bool) error {
	return s.subscriptionRepo.SetSubscriptionPlanActive(ctx, planID, active)
}
