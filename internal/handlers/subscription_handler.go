package handlers

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"

	"github.com/Neue-Konzepte-BaaS/backend/internal/middleware"
	"github.com/Neue-Konzepte-BaaS/backend/internal/models"
	"github.com/Neue-Konzepte-BaaS/backend/internal/services"
	"github.com/Neue-Konzepte-BaaS/backend/internal/webutils"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

type SubscriptionHandler struct {
	subscriptionService services.SubscriptionService
}

func NewSubscriptionHandler(subscriptionService services.SubscriptionService) *SubscriptionHandler {
	return &SubscriptionHandler{subscriptionService: subscriptionService}
}

type subscriptionPlanResponse struct {
	ID          string `json:"id"`
	Code        string `json:"code"`
	DisplayName string `json:"displayName"`
	MaxPlots    *int32 `json:"maxPlots"`
	PriceCents  int32  `json:"priceCents"`
	IsActive    bool   `json:"isActive"`
}

func toSubscriptionPlanResponse(plan models.SubscriptionPlan) subscriptionPlanResponse {
	return subscriptionPlanResponse{
		ID:          plan.ID.String(),
		Code:        string(plan.Code),
		DisplayName: plan.DisplayName,
		MaxPlots:    plan.MaxPlots,
		PriceCents:  plan.PriceCents,
		IsActive:    plan.IsActive,
	}
}

// GetPlans returns the subscription tiers currently open to new
// subscriptions. Public, same precedent as GET /api/crops: a prospective
// farmer may want to see pricing before registering.
func (h *SubscriptionHandler) GetPlans(w http.ResponseWriter, r *http.Request) {
	plans, err := h.subscriptionService.GetAvailablePlans(r.Context())
	if err != nil {
		slog.Error("getting subscription plans failed", "error", err)
		webutils.WriteError(w, http.StatusInternalServerError, "internal error")
		return
	}

	res := make([]subscriptionPlanResponse, len(plans))
	for i, plan := range plans {
		res[i] = toSubscriptionPlanResponse(plan)
	}
	webutils.WriteJSON(w, http.StatusOK, res)
}

type createSubscriptionCheckoutSessionRequest struct {
	PlanID string `json:"planId"`
}

type subscriptionCheckoutResponse struct {
	ClientSecret string `json:"clientSecret"`
	PlanCode     string `json:"planCode"`
	PriceCents   int32  `json:"priceCents"`
}

// CreateCheckoutSession opens a Stripe subscription checkout for the
// authenticated farmer's chosen plan. It must be mounted behind
// RequireAuth and RequireRole(models.RoleFarmer) -- deliberately not behind
// RequireActiveSubscription, since a farmer with no subscription yet is
// exactly who needs to reach this.
func (h *SubscriptionHandler) CreateCheckoutSession(w http.ResponseWriter, r *http.Request) {
	var req createSubscriptionCheckoutSessionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		webutils.WriteError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	planID, err := uuid.Parse(req.PlanID)
	if err != nil {
		webutils.WriteError(w, http.StatusBadRequest, "invalid plan id")
		return
	}

	claims := middleware.MustClaimsFromContext(r.Context())

	result, err := h.subscriptionService.CreateSubscriptionCheckoutSession(r.Context(), claims.UserID, planID)
	if errors.Is(err, services.ErrSubscriptionAlreadyActive) {
		webutils.WriteError(w, http.StatusConflict, "you already have a subscription")
		return
	}
	if errors.Is(err, services.ErrNotFound) {
		webutils.WriteError(w, http.StatusNotFound, "subscription plan not found")
		return
	}
	if err != nil {
		slog.Error("creating subscription checkout session failed", "error", err)
		webutils.WriteError(w, http.StatusInternalServerError, "internal error")
		return
	}

	webutils.WriteJSON(w, http.StatusCreated, subscriptionCheckoutResponse{
		ClientSecret: result.ClientSecret,
		PlanCode:     string(result.PlanCode),
		PriceCents:   result.PriceCents,
	})
}

type subscriptionStatusResponse struct {
	Status           string  `json:"status"`
	CurrentPeriodEnd *string `json:"currentPeriodEnd,omitempty"`
	PlanID           string  `json:"planId"`
	PlanCode         string  `json:"planCode"`
	PlanDisplayName  string  `json:"planDisplayName"`
	PriceCents       int32   `json:"priceCents"`
}

// toSubscriptionStatusResponse folds a subscription and its plan into the
// response shape shared by GetSubscriptionStatus and UpgradeSubscription --
// the frontend needs the plan alongside the status to know which tiers
// still count as an upgrade.
func toSubscriptionStatusResponse(sub models.FarmerSubscription, plan models.SubscriptionPlan) subscriptionStatusResponse {
	res := subscriptionStatusResponse{
		Status:          string(sub.Status),
		PlanID:          plan.ID.String(),
		PlanCode:        string(plan.Code),
		PlanDisplayName: plan.DisplayName,
		PriceCents:      plan.PriceCents,
	}
	if sub.CurrentPeriodEnd != nil {
		formatted := sub.CurrentPeriodEnd.Format("2006-01-02T15:04:05Z07:00")
		res.CurrentPeriodEnd = &formatted
	}
	return res
}

// GetSubscriptionStatus returns the authenticated farmer's own subscription
// status. It must be mounted behind RequireAuth and
// RequireRole(models.RoleFarmer), deliberately not behind
// RequireActiveSubscription for the same reason as CreateCheckoutSession.
func (h *SubscriptionHandler) GetSubscriptionStatus(w http.ResponseWriter, r *http.Request) {
	claims := middleware.MustClaimsFromContext(r.Context())

	sub, err := h.subscriptionService.GetSubscriptionStatus(r.Context(), claims.UserID)
	if errors.Is(err, services.ErrNotFound) {
		webutils.WriteError(w, http.StatusNotFound, "no subscription found")
		return
	}
	if err != nil {
		slog.Error("getting subscription status failed", "error", err)
		webutils.WriteError(w, http.StatusInternalServerError, "internal error")
		return
	}

	plan, err := h.subscriptionService.GetPlanByID(r.Context(), sub.Plan)
	if err != nil {
		slog.Error("getting subscription plan failed", "error", err)
		webutils.WriteError(w, http.StatusInternalServerError, "internal error")
		return
	}

	webutils.WriteJSON(w, http.StatusOK, toSubscriptionStatusResponse(sub, plan))
}

// ListPlansAdmin returns every subscription plan, active or retired. It
// must be mounted behind RequireAuth and RequireRole(models.RoleAdmin).
func (h *SubscriptionHandler) ListPlansAdmin(w http.ResponseWriter, r *http.Request) {
	plans, err := h.subscriptionService.ListAllPlans(r.Context())
	if err != nil {
		slog.Error("listing subscription plans failed", "error", err)
		webutils.WriteError(w, http.StatusInternalServerError, "internal error")
		return
	}

	res := make([]subscriptionPlanResponse, len(plans))
	for i, plan := range plans {
		res[i] = toSubscriptionPlanResponse(plan)
	}
	webutils.WriteJSON(w, http.StatusOK, res)
}

type updatePlanPriceRequest struct {
	PriceCents int32 `json:"priceCents"`
}

// UpdatePlanPrice creates a new Stripe Price for the plan and repoints it
// there. It must be mounted behind RequireAuth and
// RequireRole(models.RoleAdmin).
func (h *SubscriptionHandler) UpdatePlanPrice(w http.ResponseWriter, r *http.Request) {
	planID, err := uuid.Parse(chi.URLParam(r, "planID"))
	if err != nil {
		webutils.WriteError(w, http.StatusBadRequest, "invalid plan id")
		return
	}

	var req updatePlanPriceRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		webutils.WriteError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.PriceCents <= 0 {
		webutils.WriteError(w, http.StatusBadRequest, "priceCents must be positive")
		return
	}

	plan, err := h.subscriptionService.UpdatePlanPrice(r.Context(), planID, req.PriceCents)
	if errors.Is(err, services.ErrNotFound) {
		webutils.WriteError(w, http.StatusNotFound, "subscription plan not found")
		return
	}
	if err != nil {
		slog.Error("updating subscription plan price failed", "error", err)
		webutils.WriteError(w, http.StatusInternalServerError, "internal error")
		return
	}

	webutils.WriteJSON(w, http.StatusOK, toSubscriptionPlanResponse(plan))
}

type upgradeSubscriptionRequest struct {
	PlanID string `json:"planId"`
}

// UpgradeSubscription moves the authenticated farmer's subscription to a
// higher-priced plan, prorating the difference immediately. It must be
// mounted behind RequireAuth, RequireRole(models.RoleFarmer), and
// RequireActiveSubscription -- an upgrade is only meaningful for a farmer
// who already has one.
func (h *SubscriptionHandler) UpgradeSubscription(w http.ResponseWriter, r *http.Request) {
	var req upgradeSubscriptionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		webutils.WriteError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	planID, err := uuid.Parse(req.PlanID)
	if err != nil {
		webutils.WriteError(w, http.StatusBadRequest, "invalid plan id")
		return
	}

	claims := middleware.MustClaimsFromContext(r.Context())

	sub, err := h.subscriptionService.UpgradeSubscription(r.Context(), claims.UserID, planID)
	if errors.Is(err, services.ErrNotFound) {
		webutils.WriteError(w, http.StatusNotFound, "subscription or plan not found")
		return
	}
	if errors.Is(err, services.ErrNotAnUpgrade) {
		webutils.WriteError(w, http.StatusConflict, "chosen plan is not an upgrade")
		return
	}
	if err != nil {
		slog.Error("upgrading subscription failed", "error", err)
		webutils.WriteError(w, http.StatusInternalServerError, "internal error")
		return
	}

	plan, err := h.subscriptionService.GetPlanByID(r.Context(), sub.Plan)
	if err != nil {
		slog.Error("getting subscription plan failed", "error", err)
		webutils.WriteError(w, http.StatusInternalServerError, "internal error")
		return
	}

	webutils.WriteJSON(w, http.StatusOK, toSubscriptionStatusResponse(sub, plan))
}

type setPlanActiveRequest struct {
	Active bool `json:"active"`
}

// SetPlanActive retires or reactivates a subscription tier. It must be
// mounted behind RequireAuth and RequireRole(models.RoleAdmin).
func (h *SubscriptionHandler) SetPlanActive(w http.ResponseWriter, r *http.Request) {
	planID, err := uuid.Parse(chi.URLParam(r, "planID"))
	if err != nil {
		webutils.WriteError(w, http.StatusBadRequest, "invalid plan id")
		return
	}

	var req setPlanActiveRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		webutils.WriteError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if err := h.subscriptionService.SetPlanActive(r.Context(), planID, req.Active); err != nil {
		slog.Error("setting subscription plan active state failed", "error", err)
		webutils.WriteError(w, http.StatusInternalServerError, "internal error")
		return
	}

	w.WriteHeader(http.StatusOK)
}
