package repositories

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/Neue-Konzepte-BaaS/backend/internal/services"
	"github.com/stripe/stripe-go/v86"
	"github.com/stripe/stripe-go/v86/webhook"
)

type stripeGateway struct {
	client        *stripe.Client
	webhookSecret string
}

// NewStripeGateway creates a PaymentGateway backed by the real Stripe API.
func NewStripeGateway(secretKey, webhookSecret string) services.PaymentGateway {
	return &stripeGateway{client: stripe.NewClient(secretKey), webhookSecret: webhookSecret}
}

// eurCurrency is a Go constant rather than something callers pick: this
// product only serves the German market, so there is nothing to switch
// between.
const eurCurrency = string(stripe.CurrencyEUR)

func (g *stripeGateway) CreateCheckoutSession(ctx context.Context, amountCents int64, description, returnURL string) (string, string, error) {
	params := &stripe.CheckoutSessionCreateParams{
		// "elements" (Custom Checkout, Payment Element) rather than
		// "embedded_page": the latter renders Stripe's own prebuilt page in
		// a cross-origin iframe that can only be reskinned via
		// branding_settings (a handful of colors/fonts). Custom Checkout
		// gives the frontend its own <PaymentElement> to lay out and style
		// with the full Appearance API -- see stripeAppearance in
		// app/routes/customer/checkout.tsx. Requires stripe-go >= v86 (its
		// pinned API version needs to be 2026-03-25.dahlia or later, which
		// is also why this bumped from v82 -- the older SDK's hardcoded
		// API version rejected "elements" outright even though it's a
		// perfectly valid value for this account).
		UIMode:    stripe.String(string(stripe.CheckoutSessionUIModeElements)),
		Mode:      stripe.String(string(stripe.CheckoutSessionModePayment)),
		ReturnURL: stripe.String(returnURL),
		LineItems: []*stripe.CheckoutSessionCreateLineItemParams{
			{
				Quantity: stripe.Int64(1),
				PriceData: &stripe.CheckoutSessionCreateLineItemPriceDataParams{
					Currency:   stripe.String(eurCurrency),
					UnitAmount: stripe.Int64(amountCents),
					ProductData: &stripe.CheckoutSessionCreateLineItemPriceDataProductDataParams{
						Name: stripe.String(description),
					},
				},
			},
		},
	}

	session, err := g.client.V1CheckoutSessions.Create(ctx, params)
	if err != nil {
		return "", "", fmt.Errorf("creating stripe checkout session: %w", err)
	}
	return session.ID, session.ClientSecret, nil
}

// subscriptionInterval is a Go constant rather than something callers pick:
// only monthly billing exists today (see the migration's doc comment for
// why yearly is deliberately out of scope for now).
const subscriptionInterval = "month"

func (g *stripeGateway) CreateSubscriptionPrice(ctx context.Context, displayName string, amountCents int64) (string, error) {
	params := &stripe.PriceCreateParams{
		Currency:   stripe.String(eurCurrency),
		UnitAmount: stripe.Int64(amountCents),
		Recurring: &stripe.PriceCreateRecurringParams{
			Interval: stripe.String(subscriptionInterval),
		},
		ProductData: &stripe.PriceCreateProductDataParams{
			Name: stripe.String(displayName),
		},
	}

	price, err := g.client.V1Prices.Create(ctx, params)
	if err != nil {
		return "", fmt.Errorf("creating stripe subscription price: %w", err)
	}
	return price.ID, nil
}

func (g *stripeGateway) CreateSubscriptionCheckoutSession(ctx context.Context, stripePriceID, customerEmail string, existingStripeCustomerID *string, returnURL string) (string, string, string, error) {
	params := &stripe.CheckoutSessionCreateParams{
		// Same UIMode choice as CreateCheckoutSession, for the same reason:
		// a frontend-styled <PaymentElement> rather than Stripe's own page.
		UIMode:    stripe.String(string(stripe.CheckoutSessionUIModeElements)),
		Mode:      stripe.String(string(stripe.CheckoutSessionModeSubscription)),
		ReturnURL: stripe.String(returnURL),
		LineItems: []*stripe.CheckoutSessionCreateLineItemParams{
			{
				Price:    stripe.String(stripePriceID),
				Quantity: stripe.Int64(1),
			},
		},
	}
	if existingStripeCustomerID != nil {
		params.Customer = stripe.String(*existingStripeCustomerID)
	} else {
		params.CustomerEmail = stripe.String(customerEmail)
	}

	session, err := g.client.V1CheckoutSessions.Create(ctx, params)
	if err != nil {
		return "", "", "", fmt.Errorf("creating stripe subscription checkout session: %w", err)
	}

	stripeCustomerID := ""
	if existingStripeCustomerID != nil {
		stripeCustomerID = *existingStripeCustomerID
	} else if session.Customer != nil {
		stripeCustomerID = session.Customer.ID
	}

	return session.ID, session.ClientSecret, stripeCustomerID, nil
}

// prorationBehaviorCreateProrations is a Go constant rather than something
// callers pick: every upgrade in this product bills the prorated difference
// immediately, so there is nothing to switch between.
const prorationBehaviorCreateProrations = "create_prorations"

func (g *stripeGateway) UpdateSubscriptionPrice(ctx context.Context, stripeSubscriptionID, newStripePriceID string) (time.Time, error) {
	sub, err := g.client.V1Subscriptions.Retrieve(ctx, stripeSubscriptionID, nil)
	if err != nil {
		return time.Time{}, fmt.Errorf("retrieving stripe subscription: %w", err)
	}
	if sub.Items == nil || len(sub.Items.Data) == 0 {
		return time.Time{}, fmt.Errorf("stripe subscription %s has no items", stripeSubscriptionID)
	}
	itemID := sub.Items.Data[0].ID

	updated, err := g.client.V1Subscriptions.Update(ctx, stripeSubscriptionID, &stripe.SubscriptionUpdateParams{
		Items: []*stripe.SubscriptionUpdateItemParams{
			{
				ID:    stripe.String(itemID),
				Price: stripe.String(newStripePriceID),
			},
		},
		ProrationBehavior: stripe.String(prorationBehaviorCreateProrations),
	})
	if err != nil {
		return time.Time{}, fmt.Errorf("updating stripe subscription price: %w", err)
	}
	if updated.Items == nil || len(updated.Items.Data) == 0 {
		return time.Time{}, fmt.Errorf("updated stripe subscription %s has no items", stripeSubscriptionID)
	}

	return time.Unix(updated.Items.Data[0].CurrentPeriodEnd, 0), nil
}

func (g *stripeGateway) CancelSubscription(ctx context.Context, stripeSubscriptionID string) error {
	_, err := g.client.V1Subscriptions.Cancel(ctx, stripeSubscriptionID, nil)
	if err != nil {
		return fmt.Errorf("canceling stripe subscription: %w", err)
	}
	return nil
}

func (g *stripeGateway) RefundCheckoutSession(ctx context.Context, sessionID string) error {
	session, err := g.client.V1CheckoutSessions.Retrieve(ctx, sessionID, nil)
	if err != nil {
		return fmt.Errorf("retrieving stripe checkout session: %w", err)
	}
	if session.PaymentIntent == nil {
		return fmt.Errorf("checkout session %s has no payment intent to refund", sessionID)
	}

	_, err = g.client.V1Refunds.Create(ctx, &stripe.RefundCreateParams{
		PaymentIntent: stripe.String(session.PaymentIntent.ID),
	})
	if err != nil {
		return fmt.Errorf("refunding stripe payment intent: %w", err)
	}
	return nil
}

// eventDataString reads a top-level string field from an untyped webhook
// payload, e.g. event.Data.Object["id"].
func eventDataString(object map[string]any, key string) (string, bool) {
	v, ok := object[key].(string)
	return v, ok && v != ""
}

func (g *stripeGateway) ParseWebhookEvent(payload []byte, sigHeader string) (services.WebhookEvent, bool, error) {
	// IgnoreAPIVersionMismatch: stripe-go's pinned API version now matches
	// this account's, so this is a forward-compatibility safety net rather
	// than a live workaround (kept in case either drifts again). Safe here
	// specifically because event.Data.Object is untyped JSON (map[string]any)
	// and we only ever read a handful of plain scalar fields below -- stable
	// across every API version -- rather than deserializing into a
	// version-specific struct that could actually be mismatched.
	event, err := webhook.ConstructEventWithOptions(payload, sigHeader, g.webhookSecret, webhook.ConstructEventOptions{
		IgnoreAPIVersionMismatch: true,
	})
	if err != nil {
		return services.WebhookEvent{}, false, fmt.Errorf("%w: %w", services.ErrInvalidWebhookSignature, err)
	}

	switch event.Type {
	case stripe.EventTypeCheckoutSessionCompleted, stripe.EventTypeCheckoutSessionExpired:
		return g.parseCheckoutSessionEvent(event)
	case stripe.EventTypeCustomerSubscriptionDeleted:
		return g.parseSubscriptionLifecycleEvent(event, services.WebhookEventSubscriptionDeleted)
	case stripe.EventTypeInvoicePaymentFailed:
		return g.parseInvoiceEvent(event, services.WebhookEventInvoicePaymentFailed)
	case stripe.EventTypeInvoicePaymentSucceeded:
		return g.parseInvoiceEvent(event, services.WebhookEventInvoicePaymentSucceeded)
	default:
		return services.WebhookEvent{}, false, nil
	}
}

// parseCheckoutSessionEvent handles checkout.session.completed/expired,
// which fire for both payment-mode and subscription-mode sessions under the
// same Stripe event type -- Stripe only distinguishes them via the
// session's own "mode" field, which this reads to pick the right
// WebhookEventType so callers never need to inspect mode themselves.
func (g *stripeGateway) parseCheckoutSessionEvent(event stripe.Event) (services.WebhookEvent, bool, error) {
	sessionID, ok := eventDataString(event.Data.Object, "id")
	if !ok {
		return services.WebhookEvent{}, false, errors.New("stripe webhook event has no checkout session id")
	}
	mode, _ := eventDataString(event.Data.Object, "mode")

	var eventType services.WebhookEventType
	switch {
	case event.Type == stripe.EventTypeCheckoutSessionCompleted && mode == string(stripe.CheckoutSessionModeSubscription):
		eventType = services.WebhookEventSubscriptionCheckoutCompleted
	case event.Type == stripe.EventTypeCheckoutSessionCompleted:
		eventType = services.WebhookEventCheckoutCompleted
	case event.Type == stripe.EventTypeCheckoutSessionExpired && mode == string(stripe.CheckoutSessionModeSubscription):
		eventType = services.WebhookEventSubscriptionCheckoutExpired
	default:
		eventType = services.WebhookEventCheckoutExpired
	}

	result := services.WebhookEvent{Type: eventType, CheckoutSessionID: sessionID}

	// A completed subscription-mode session already has a live Stripe
	// Subscription by the time this fires. Verified against a real webhook
	// payload (API version 2025-09-30.clover): unlike an *expanded* object
	// reference, an unexpanded one here is a plain string id ("sub_..."),
	// not a nested {"id": "..."} object -- same shape as
	// event.Data.Object["customer"] ("cus_...").
	if eventType == services.WebhookEventSubscriptionCheckoutCompleted {
		if id, ok := eventDataString(event.Data.Object, "subscription"); ok {
			result.StripeSubscriptionID = id
		}
	}

	return result, true, nil
}

func (g *stripeGateway) parseSubscriptionLifecycleEvent(event stripe.Event, eventType services.WebhookEventType) (services.WebhookEvent, bool, error) {
	subscriptionID, ok := eventDataString(event.Data.Object, "id")
	if !ok {
		return services.WebhookEvent{}, false, errors.New("stripe webhook event has no subscription id")
	}
	return services.WebhookEvent{Type: eventType, StripeSubscriptionID: subscriptionID}, true, nil
}

// parseInvoiceEvent handles invoice.payment_failed/invoice.payment_succeeded.
// NOTE: the exact field this account's pinned API version uses to link an
// invoice back to its subscription (and, for a succeeded payment, the new
// current_period_end) should be confirmed against a real payload via
// `stripe trigger`/the CLI during testing -- Stripe has moved these fields
// across API versions (a top-level "subscription" field on older versions;
// nested under "parent.subscription_details.subscription" on newer ones),
// so both shapes are checked here defensively.
func (g *stripeGateway) parseInvoiceEvent(event stripe.Event, eventType services.WebhookEventType) (services.WebhookEvent, bool, error) {
	subscriptionID, ok := eventDataString(event.Data.Object, "subscription")
	if !ok {
		if parent, isMap := event.Data.Object["parent"].(map[string]any); isMap {
			if details, isMap := parent["subscription_details"].(map[string]any); isMap {
				subscriptionID, ok = eventDataString(details, "subscription")
			}
		}
	}
	if !ok {
		return services.WebhookEvent{}, false, errors.New("stripe webhook event has no subscription id")
	}

	result := services.WebhookEvent{Type: eventType, StripeSubscriptionID: subscriptionID}

	if lines, isMap := event.Data.Object["lines"].(map[string]any); isMap {
		if data, isSlice := lines["data"].([]any); isSlice && len(data) > 0 {
			if line, isMap := data[0].(map[string]any); isMap {
				if period, isMap := line["period"].(map[string]any); isMap {
					if end, isNum := period["end"].(float64); isNum {
						t := time.Unix(int64(end), 0)
						result.CurrentPeriodEnd = &t
					}
				}
			}
		}
	}

	return result, true, nil
}
