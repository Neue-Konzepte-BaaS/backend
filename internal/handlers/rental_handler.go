package handlers

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"time"

	"github.com/Neue-Konzepte-BaaS/backend/internal/middleware"
	"github.com/Neue-Konzepte-BaaS/backend/internal/models"
	"github.com/Neue-Konzepte-BaaS/backend/internal/services"
	"github.com/Neue-Konzepte-BaaS/backend/internal/webutils"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

type RentalHandler struct {
	rentalService  services.RentalService
	paymentService services.PaymentService
}

func NewRentalHandler(rentalService services.RentalService, paymentService services.PaymentService) *RentalHandler {
	return &RentalHandler{rentalService: rentalService, paymentService: paymentService}
}

type rentalResponse struct {
	ID        string  `json:"id"`
	PlotID    string  `json:"plotId"`
	CropID    string  `json:"cropId"`
	StartAt   string  `json:"startAt"`
	EndAt     string  `json:"endAt"`
	Status    string  `json:"status"`
	Message   string  `json:"message"`
	DecidedAt *string `json:"decidedAt"`
}

type rentalWithPlotResponse struct {
	rentalResponse
	Plot plotResponse `json:"plot"`
	Crop cropResponse `json:"crop"`
}

type customerResponse struct {
	ID        string `json:"id"`
	Email     string `json:"email"`
	FirstName string `json:"firstName"`
	LastName  string `json:"lastName"`
}

type rentalWithPlotAndCustomerResponse struct {
	rentalResponse
	Plot      plotResponse     `json:"plot"`
	FieldName string           `json:"fieldName"`
	Customer  customerResponse `json:"customer"`
}

// ApproveRental approves a still-requested rental on one of the
// authenticated farmer's own plots. It must be mounted behind RequireAuth
// and RequireRole(models.RoleFarmer).
func (h *RentalHandler) ApproveRental(w http.ResponseWriter, r *http.Request) {
	h.decideRental(w, r, func(ctx context.Context, farmer, rentalID uuid.UUID) (models.Rental, error) {
		return h.rentalService.ApproveRental(ctx, farmer, rentalID)
	})
}

// DeclineRental declines a still-requested rental on one of the
// authenticated farmer's own plots, freeing the plot for that period. If
// the rental had already been paid for, the payment is refunded. It must be
// mounted behind RequireAuth and RequireRole(models.RoleFarmer).
func (h *RentalHandler) DeclineRental(w http.ResponseWriter, r *http.Request) {
	h.decideRental(w, r, func(ctx context.Context, farmer, rentalID uuid.UUID) (models.Rental, error) {
		rental, err := h.rentalService.DeclineRental(ctx, farmer, rentalID)
		if err != nil {
			return models.Rental{}, err
		}
		// The decline itself already succeeded and the plot is freed; a
		// refund failure must not undo that. It is logged so it can be
		// chased up out of band instead.
		if err := h.paymentService.RefundIfPaid(ctx, rental.ID); err != nil {
			slog.Error("refunding declined rental failed", "rental", rental.ID, "error", err)
		}
		return rental, nil
	})
}

func (h *RentalHandler) decideRental(w http.ResponseWriter, r *http.Request, decide func(context.Context, uuid.UUID, uuid.UUID) (models.Rental, error)) {
	rentalID, err := uuid.Parse(chi.URLParam(r, "rentalID"))
	if err != nil {
		webutils.WriteError(w, http.StatusBadRequest, "invalid rental id")
		return
	}

	claims := middleware.MustClaimsFromContext(r.Context())

	rental, err := decide(r.Context(), claims.UserID, rentalID)
	if errors.Is(err, services.ErrForbidden) {
		webutils.WriteError(w, http.StatusForbidden, "plot is not owned by this farmer")
		return
	}
	if errors.Is(err, services.ErrNotFound) {
		webutils.WriteError(w, http.StatusNotFound, "rental not found")
		return
	}
	if errors.Is(err, services.ErrRentalAlreadyDecided) {
		webutils.WriteError(w, http.StatusConflict, "rental request was already decided")
		return
	}
	if err != nil {
		slog.Error("deciding rental failed", "error", err)
		webutils.WriteError(w, http.StatusInternalServerError, "internal error")
		return
	}

	webutils.WriteJSON(w, http.StatusOK, toRentalResponse(rental))
}

// GetRentals returns the authenticated customer's own rentals. It must be
// mounted behind RequireAuth and RequireRole(models.RoleCustomer).
func (h *RentalHandler) GetRentals(w http.ResponseWriter, r *http.Request) {
	claims := middleware.MustClaimsFromContext(r.Context())

	rentals, err := h.rentalService.GetRentals(r.Context(), claims.UserID)
	if err != nil {
		slog.Error("getting rentals failed", "error", err)
		webutils.WriteError(w, http.StatusInternalServerError, "internal error")
		return
	}

	res := make([]rentalWithPlotResponse, len(rentals))
	for i, rental := range rentals {
		res[i] = rentalWithPlotResponse{
			rentalResponse: toRentalResponse(rental.Rental),
			Plot: plotResponse{
				ID:               rental.Plot.ID.String(),
				Name:             rental.Plot.Name,
				Field:            rental.Plot.Field.String(),
				Coordinates:      encodePolygon(rental.Plot.Coordinates),
				AreaSquareMeters: rental.Plot.AreaSquareMeters,
			},
			Crop: toCropResponse(rental.Crop),
		}
	}

	webutils.WriteJSON(w, http.StatusOK, res)
}

// GetFarmRentals returns every rental on the authenticated farmer's own
// plots, active and historic. It must be mounted behind RequireAuth and
// RequireRole(models.RoleFarmer).
func (h *RentalHandler) GetFarmRentals(w http.ResponseWriter, r *http.Request) {
	claims := middleware.MustClaimsFromContext(r.Context())

	rentals, err := h.rentalService.GetRentalsForFarmer(r.Context(), claims.UserID)
	if err != nil {
		slog.Error("getting farm rentals failed", "error", err)
		webutils.WriteError(w, http.StatusInternalServerError, "internal error")
		return
	}

	res := make([]rentalWithPlotAndCustomerResponse, len(rentals))
	for i, rental := range rentals {
		res[i] = rentalWithPlotAndCustomerResponse{
			rentalResponse: toRentalResponse(rental.Rental),
			Plot: plotResponse{
				ID:               rental.Plot.ID.String(),
				Name:             rental.Plot.Name,
				Field:            rental.Plot.Field.String(),
				Coordinates:      encodePolygon(rental.Plot.Coordinates),
				AreaSquareMeters: rental.Plot.AreaSquareMeters,
			},
			FieldName: rental.FieldName,
			Customer: customerResponse{
				ID:        rental.Customer.AccountID.String(),
				Email:     rental.Customer.Email,
				FirstName: rental.Customer.FirstName,
				LastName:  rental.Customer.LastName,
			},
		}
	}

	webutils.WriteJSON(w, http.StatusOK, res)
}

func toRentalResponse(rental models.Rental) rentalResponse {
	var decidedAt *string
	if rental.DecidedAt != nil {
		s := rental.DecidedAt.Format(time.RFC3339)
		decidedAt = &s
	}
	return rentalResponse{
		ID:        rental.ID.String(),
		PlotID:    rental.PlotID.String(),
		CropID:    rental.CropID.String(),
		StartAt:   rental.StartAt.Format(time.RFC3339),
		EndAt:     rental.EndAt.Format(time.RFC3339),
		Status:    string(rental.Status),
		Message:   rental.Message,
		DecidedAt: decidedAt,
	}
}
