package repositories

import (
	"context"
	"errors"
	"fmt"

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

func (g *stripeGateway) ParseWebhookEvent(payload []byte, sigHeader string) (services.WebhookEvent, bool, error) {
	// IgnoreAPIVersionMismatch: stripe-go's pinned API version now matches
	// this account's, so this is a forward-compatibility safety net rather
	// than a live workaround (kept in case either drifts again). Safe here
	// specifically because event.Data.Object is untyped JSON (map[string]any)
	// and we only ever read its "id" field below -- a plain string, stable
	// across every API version -- rather than deserializing into a
	// version-specific struct that could actually be mismatched.
	event, err := webhook.ConstructEventWithOptions(payload, sigHeader, g.webhookSecret, webhook.ConstructEventOptions{
		IgnoreAPIVersionMismatch: true,
	})
	if err != nil {
		return services.WebhookEvent{}, false, fmt.Errorf("%w: %w", services.ErrInvalidWebhookSignature, err)
	}

	var eventType services.WebhookEventType
	switch event.Type {
	case stripe.EventTypeCheckoutSessionCompleted:
		eventType = services.WebhookEventCheckoutCompleted
	case stripe.EventTypeCheckoutSessionExpired:
		eventType = services.WebhookEventCheckoutExpired
	default:
		return services.WebhookEvent{}, false, nil
	}

	sessionID, ok := event.Data.Object["id"].(string)
	if !ok || sessionID == "" {
		return services.WebhookEvent{}, false, errors.New("stripe webhook event has no checkout session id")
	}

	return services.WebhookEvent{Type: eventType, CheckoutSessionID: sessionID}, true, nil
}
