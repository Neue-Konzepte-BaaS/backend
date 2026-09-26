package services

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/Neue-Konzepte-BaaS/backend/internal/models"
	"github.com/google/uuid"
)

// checkoutReturnPath is where Stripe redirects the top-level page once
// payment completes inside the embedded iframe. {CHECKOUT_SESSION_ID} is a
// literal placeholder Stripe itself resolves.
const checkoutReturnPath = "/customer/payment/return?session_id={CHECKOUT_SESSION_ID}"

type PaymentService interface {
	// CreateCheckoutSession validates the plot/crop/startAt/message exactly
	// like RentalService.RequestRental would -- crop exists, plot exists,
	// the plot offers that crop and both halves of its price are set, the
	// start date falls in the allowed window -- minus the final insert,
	// plus a fast-fail availability check, so a customer who could not book
	// anyway never reaches Stripe. On success it opens a Stripe Embedded
	// Checkout session for the computed price and records a Pending
	// rental_checkout row. No rental exists yet: one is only ever created
	// once the checkout.session.completed webhook confirms payment.
	CreateCheckoutSession(ctx context.Context, customer, plot, crop uuid.UUID, startAt time.Time, message string) (models.CheckoutSessionResult, error)
	// GetCheckoutSessionStatus returns the status of the caller's own
	// checkout session (identified by its Stripe session id), and the
	// rental it produced once one exists. Returns ErrNotFound if the
	// session does not exist, and ErrForbidden if it belongs to a different
	// customer.
	GetCheckoutSessionStatus(ctx context.Context, customer uuid.UUID, stripeSessionID string) (models.CheckoutSessionStatusResult, error)
	// HandleWebhookEvent verifies and processes one Stripe webhook
	// delivery. It is idempotent: Stripe retries delivery, so processing
	// the same event twice must be a no-op the second time.
	HandleWebhookEvent(ctx context.Context, payload []byte, sigHeader string) error
	// RefundIfPaid refunds and marks Refunded the completed checkout that
	// produced rental, if any. It is a no-op, not an error, when rental was
	// never paid for through Stripe (e.g. it predates this feature).
	RefundIfPaid(ctx context.Context, rental uuid.UUID) error
}

type paymentService struct {
	rentalService  RentalService
	rentalRepo     RentalRepository
	checkoutRepo   RentalCheckoutRepository
	paymentGateway PaymentGateway
	plotRepo       PlotRepository
	cropRepo       CropRepository
	fieldRepo      FieldRepository
	farmRepo       FarmRepository
	frontendURL    string
}

func NewPaymentService(rentalService RentalService, rentalRepo RentalRepository, checkoutRepo RentalCheckoutRepository, paymentGateway PaymentGateway, plotRepo PlotRepository, cropRepo CropRepository, fieldRepo FieldRepository, farmRepo FarmRepository, frontendURL string) PaymentService {
	return &paymentService{
		rentalService:  rentalService,
		rentalRepo:     rentalRepo,
		checkoutRepo:   checkoutRepo,
		paymentGateway: paymentGateway,
		plotRepo:       plotRepo,
		cropRepo:       cropRepo,
		fieldRepo:      fieldRepo,
		farmRepo:       farmRepo,
		frontendURL:    frontendURL,
	}
}

func (s *paymentService) CreateCheckoutSession(ctx context.Context, customer, plot, crop uuid.UUID, startAt time.Time, message string) (models.CheckoutSessionResult, error) {
	if strings.TrimSpace(message) == "" {
		return models.CheckoutSessionResult{}, ErrInvalidRentalRequest
	}
	notice := time.Until(startAt)
	if notice < minRentalNotice || notice > maxRentalNotice {
		return models.CheckoutSessionResult{}, ErrInvalidRentalRequest
	}

	cropDetails, err := s.cropRepo.GetCropByID(ctx, crop)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			return models.CheckoutSessionResult{}, err
		}
		return models.CheckoutSessionResult{}, fmt.Errorf("getting crop: %w", err)
	}

	plotDetails, err := s.plotRepo.GetPlotByID(ctx, plot)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			return models.CheckoutSessionResult{}, err
		}
		return models.CheckoutSessionResult{}, fmt.Errorf("getting plot: %w", err)
	}

	offeredCrops, err := s.cropRepo.GetCropsByPlot(ctx, plot)
	if err != nil {
		return models.CheckoutSessionResult{}, fmt.Errorf("getting plot crops: %w", err)
	}
	if !cropOffered(offeredCrops, crop) {
		return models.CheckoutSessionResult{}, ErrCropNotOffered
	}

	// A crop the plot lists but that is missing either half of its price is
	// not actually rentable -- from a customer's perspective that is
	// indistinguishable from not being offered at all, so it reuses the
	// same sentinel (and thus the same "crop is not offered by this plot"
	// message) rather than a new, more specific one.
	if plotDetails.BasePriceCentsPerSqmPerWeek == nil {
		return models.CheckoutSessionResult{}, ErrCropNotOffered
	}
	farmID, err := s.fieldRepo.GetFieldFarm(ctx, plotDetails.Field)
	if err != nil {
		return models.CheckoutSessionResult{}, fmt.Errorf("looking up field farm: %w", err)
	}
	farmCropRate, err := s.farmRepo.GetFarmCropRate(ctx, farmID, crop)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			return models.CheckoutSessionResult{}, ErrCropNotOffered
		}
		return models.CheckoutSessionResult{}, fmt.Errorf("getting farm crop rate: %w", err)
	}

	// A fast-fail check only: rental_no_overlap on the rental table is what
	// actually prevents a double booking, at RequestRental time in the
	// webhook. This just saves a customer a trip through Stripe for a plot
	// that is obviously already taken.
	available, err := s.rentalRepo.IsPlotAvailable(ctx, plot, startAt, cropDetails.DurationMonths)
	if err != nil {
		return models.CheckoutSessionResult{}, fmt.Errorf("checking plot availability: %w", err)
	}
	if !available {
		return models.CheckoutSessionResult{}, ErrPlotUnavailable
	}

	priceCents := ComputeRentalPriceCents(*plotDetails.BasePriceCentsPerSqmPerWeek, farmCropRate, plotDetails.AreaSquareMeters, cropDetails.DurationMonths)
	description := fmt.Sprintf("%s — %s", plotDetails.Name, cropDetails.Name)
	returnURL := s.frontendURL + checkoutReturnPath

	sessionID, clientSecret, err := s.paymentGateway.CreateCheckoutSession(ctx, int64(priceCents), description, returnURL)
	if err != nil {
		return models.CheckoutSessionResult{}, fmt.Errorf("creating stripe checkout session: %w", err)
	}

	if _, err := s.checkoutRepo.CreateCheckout(ctx, models.RentalCheckout{
		Customer:                customer,
		Plot:                    plot,
		Crop:                    crop,
		StartAt:                 startAt,
		Message:                 message,
		StripeCheckoutSessionID: sessionID,
		AmountCents:             priceCents,
	}); err != nil {
		return models.CheckoutSessionResult{}, fmt.Errorf("recording checkout: %w", err)
	}

	return models.CheckoutSessionResult{
		ClientSecret: clientSecret,
		PlotName:     plotDetails.Name,
		CropName:     cropDetails.Name,
		PriceCents:   priceCents,
	}, nil
}

func (s *paymentService) GetCheckoutSessionStatus(ctx context.Context, customer uuid.UUID, stripeSessionID string) (models.CheckoutSessionStatusResult, error) {
	checkout, err := s.checkoutRepo.GetCheckoutBySessionID(ctx, stripeSessionID)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			return models.CheckoutSessionStatusResult{}, err
		}
		return models.CheckoutSessionStatusResult{}, fmt.Errorf("getting checkout: %w", err)
	}
	if checkout.Customer != customer {
		return models.CheckoutSessionStatusResult{}, ErrForbidden
	}

	result := models.CheckoutSessionStatusResult{Status: checkout.Status}
	if checkout.Rental != nil {
		rental, err := s.rentalRepo.GetRentalByID(ctx, *checkout.Rental)
		if err != nil {
			return models.CheckoutSessionStatusResult{}, fmt.Errorf("getting rental: %w", err)
		}
		result.Rental = &rental
	}
	return result, nil
}

func (s *paymentService) HandleWebhookEvent(ctx context.Context, payload []byte, sigHeader string) error {
	event, ok, err := s.paymentGateway.ParseWebhookEvent(payload, sigHeader)
	if err != nil {
		return err
	}
	if !ok {
		return nil // an event type this codebase does not act on
	}

	checkout, err := s.checkoutRepo.GetCheckoutBySessionID(ctx, event.CheckoutSessionID)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			slog.Warn("stripe webhook for unknown checkout session", "sessionId", event.CheckoutSessionID)
			return nil
		}
		return fmt.Errorf("looking up checkout: %w", err)
	}

	switch event.Type {
	case WebhookEventCheckoutExpired:
		if _, err := s.checkoutRepo.ExpireCheckout(ctx, checkout.ID); err != nil && !errors.Is(err, ErrCheckoutAlreadyProcessed) {
			return fmt.Errorf("expiring checkout: %w", err)
		}
		return nil
	case WebhookEventCheckoutCompleted:
		return s.completeCheckout(ctx, checkout)
	}
	return nil
}

// completeCheckout creates the rental request a confirmed payment is for.
// It is guarded by checkout.Status so a retried webhook delivery for an
// already-processed session is a no-op rather than creating a second
// rental request for the same payment.
func (s *paymentService) completeCheckout(ctx context.Context, checkout models.RentalCheckout) error {
	if checkout.Status != models.CheckoutStatusPending {
		return nil
	}

	rental, err := s.rentalService.RequestRental(ctx, checkout.Customer, checkout.Plot, checkout.Crop, checkout.StartAt, checkout.Message)
	if err != nil {
		if errors.Is(err, ErrPlotUnavailable) || errors.Is(err, ErrCropNotOffered) || errors.Is(err, ErrNotFound) || errors.Is(err, ErrInvalidRentalRequest) {
			// The money was already captured by Stripe but no rental could
			// be created -- refund it rather than leave the customer
			// charged for nothing. ErrPlotUnavailable is the realistic
			// case: two customers paid for an overlapping period and only
			// one insert can win the exclusion constraint. The others are
			// unlikely (e.g. the crop was un-offered in between) but
			// handled the same way defensively.
			if refundErr := s.paymentGateway.RefundCheckoutSession(ctx, checkout.StripeCheckoutSessionID); refundErr != nil {
				return fmt.Errorf("refunding after failed rental creation: %w", refundErr)
			}
			if _, failErr := s.checkoutRepo.FailCheckout(ctx, checkout.ID); failErr != nil && !errors.Is(failErr, ErrCheckoutAlreadyProcessed) {
				return fmt.Errorf("marking checkout failed: %w", failErr)
			}
			return nil
		}
		return fmt.Errorf("creating rental request: %w", err)
	}

	if _, err := s.checkoutRepo.CompleteCheckout(ctx, checkout.ID, rental.ID); err != nil && !errors.Is(err, ErrCheckoutAlreadyProcessed) {
		return fmt.Errorf("completing checkout: %w", err)
	}
	return nil
}

func (s *paymentService) RefundIfPaid(ctx context.Context, rental uuid.UUID) error {
	checkout, err := s.checkoutRepo.GetCompletedCheckoutByRental(ctx, rental)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			return nil
		}
		return fmt.Errorf("looking up checkout: %w", err)
	}

	if err := s.paymentGateway.RefundCheckoutSession(ctx, checkout.StripeCheckoutSessionID); err != nil {
		return fmt.Errorf("refunding via stripe: %w", err)
	}
	if _, err := s.checkoutRepo.MarkCheckoutRefunded(ctx, checkout.ID); err != nil && !errors.Is(err, ErrCheckoutAlreadyProcessed) {
		return fmt.Errorf("marking checkout refunded: %w", err)
	}
	return nil
}
