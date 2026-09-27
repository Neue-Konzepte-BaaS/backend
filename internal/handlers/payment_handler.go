package handlers

import (
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"time"

	"github.com/Neue-Konzepte-BaaS/backend/internal/middleware"
	"github.com/Neue-Konzepte-BaaS/backend/internal/services"
	"github.com/Neue-Konzepte-BaaS/backend/internal/webutils"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

type PaymentHandler struct {
	paymentService services.PaymentService
}

func NewPaymentHandler(paymentService services.PaymentService) *PaymentHandler {
	return &PaymentHandler{paymentService: paymentService}
}

// maxWebhookBodyBytes bounds how much of a Stripe webhook request body is
// read: real payloads are a few KB, this is a generous ceiling against a
// malformed or hostile request, not a realistic limit.
const maxWebhookBodyBytes = 1 << 20 // 1 MiB

type createCheckoutSessionRequest struct {
	PlotID  string `json:"plotId"`
	CropID  string `json:"cropId"`
	StartAt string `json:"startAt"`
	Message string `json:"message"`
}

type checkoutSessionResponse struct {
	ClientSecret string `json:"clientSecret"`
	PlotName     string `json:"plotName"`
	CropName     string `json:"cropName"`
	PriceCents   int32  `json:"priceCents"`
}

type checkoutSessionStatusResponse struct {
	Status string          `json:"status"`
	Rental *rentalResponse `json:"rental,omitempty"`
}

// CreateCheckoutSession opens a Stripe Embedded Checkout session for the
// authenticated customer to pay for a plot/crop rental. No rental is
// created here -- that only happens once Stripe confirms the payment via
// webhook. It must be mounted behind RequireAuth and
// RequireRole(models.RoleCustomer).
func (h *PaymentHandler) CreateCheckoutSession(w http.ResponseWriter, r *http.Request) {
	var req createCheckoutSessionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		webutils.WriteError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	plotID, err := uuid.Parse(req.PlotID)
	if err != nil {
		webutils.WriteError(w, http.StatusBadRequest, "invalid plot id")
		return
	}

	cropID, err := uuid.Parse(req.CropID)
	if err != nil {
		webutils.WriteError(w, http.StatusBadRequest, "invalid crop id")
		return
	}

	startAt, err := time.Parse(time.RFC3339, req.StartAt)
	if err != nil {
		webutils.WriteError(w, http.StatusBadRequest, "invalid start date")
		return
	}

	claims := middleware.MustClaimsFromContext(r.Context())

	result, err := h.paymentService.CreateCheckoutSession(r.Context(), claims.UserID, plotID, cropID, startAt, req.Message)
	if errors.Is(err, services.ErrInvalidRentalRequest) {
		webutils.WriteError(w, http.StatusBadRequest, "start date must be 1 to 60 days from now and message must not be blank")
		return
	}
	if errors.Is(err, services.ErrNotFound) {
		webutils.WriteError(w, http.StatusNotFound, "plot or crop not found")
		return
	}
	if errors.Is(err, services.ErrCropNotOffered) {
		webutils.WriteError(w, http.StatusConflict, "crop is not offered by this plot")
		return
	}
	if errors.Is(err, services.ErrPlotUnavailable) {
		webutils.WriteError(w, http.StatusConflict, "plot is already rented")
		return
	}
	if err != nil {
		slog.Error("creating checkout session failed", "error", err)
		webutils.WriteError(w, http.StatusInternalServerError, "internal error")
		return
	}

	webutils.WriteJSON(w, http.StatusCreated, checkoutSessionResponse{
		ClientSecret: result.ClientSecret,
		PlotName:     result.PlotName,
		CropName:     result.CropName,
		PriceCents:   result.PriceCents,
	})
}

// GetCheckoutSessionStatus returns the status of a checkout session
// belonging to the authenticated customer, identified by its Stripe session
// id. It must be mounted behind RequireAuth and
// RequireRole(models.RoleCustomer).
func (h *PaymentHandler) GetCheckoutSessionStatus(w http.ResponseWriter, r *http.Request) {
	sessionID := chi.URLParam(r, "sessionID")

	claims := middleware.MustClaimsFromContext(r.Context())

	result, err := h.paymentService.GetCheckoutSessionStatus(r.Context(), claims.UserID, sessionID)
	if errors.Is(err, services.ErrNotFound) {
		webutils.WriteError(w, http.StatusNotFound, "checkout session not found")
		return
	}
	if errors.Is(err, services.ErrForbidden) {
		webutils.WriteError(w, http.StatusForbidden, "checkout session does not belong to you")
		return
	}
	if err != nil {
		slog.Error("getting checkout session status failed", "error", err)
		webutils.WriteError(w, http.StatusInternalServerError, "internal error")
		return
	}

	res := checkoutSessionStatusResponse{Status: string(result.Status)}
	if result.Rental != nil {
		rental := toRentalResponse(*result.Rental)
		res.Rental = &rental
	}
	webutils.WriteJSON(w, http.StatusOK, res)
}

// HandleStripeWebhook processes a Stripe webhook delivery. It is a public
// endpoint (no auth required) -- Stripe itself is the caller, authenticated
// instead by the Stripe-Signature header. It must be mounted without
// RequireAuth.
func (h *PaymentHandler) HandleStripeWebhook(w http.ResponseWriter, r *http.Request) {
	payload, err := io.ReadAll(io.LimitReader(r.Body, maxWebhookBodyBytes))
	if err != nil {
		webutils.WriteError(w, http.StatusBadRequest, "could not read request body")
		return
	}

	err = h.paymentService.HandleWebhookEvent(r.Context(), payload, r.Header.Get("Stripe-Signature"))
	if errors.Is(err, services.ErrInvalidWebhookSignature) {
		webutils.WriteError(w, http.StatusBadRequest, "invalid webhook signature")
		return
	}
	if err != nil {
		// A non-2xx response makes Stripe retry delivery later, which is
		// what we want for an error that might be transient.
		slog.Error("handling stripe webhook failed", "error", err)
		webutils.WriteError(w, http.StatusInternalServerError, "internal error")
		return
	}

	w.WriteHeader(http.StatusOK)
}
