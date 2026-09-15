package handlers

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"time"

	"github.com/Neue-Konzepte-BaaS/backend/internal/middleware"
	"github.com/Neue-Konzepte-BaaS/backend/internal/models"
	"github.com/Neue-Konzepte-BaaS/backend/internal/services"
	"github.com/Neue-Konzepte-BaaS/backend/internal/webutils"
	"github.com/google/uuid"
)

type RentalHandler struct {
	rentalService services.RentalService
}

func NewRentalHandler(rentalService services.RentalService) *RentalHandler {
	return &RentalHandler{rentalService: rentalService}
}

type rentPlotRequest struct {
	PlotID string `json:"plotId"`
	CropID string `json:"cropId"`
}

type rentalResponse struct {
	ID      string `json:"id"`
	PlotID  string `json:"plotId"`
	CropID  string `json:"cropId"`
	StartAt string `json:"startAt"`
	EndAt   string `json:"endAt"`
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

// RentPlot books a plot for the authenticated customer. It must be mounted
// behind RequireAuth and RequireRole(models.RoleCustomer).
func (h *RentalHandler) RentPlot(w http.ResponseWriter, r *http.Request) {
	var req rentPlotRequest
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

	claims := middleware.MustClaimsFromContext(r.Context())

	rental, err := h.rentalService.RentPlot(r.Context(), claims.UserID, plotID, cropID)
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
		slog.Error("renting plot failed", "error", err)
		webutils.WriteError(w, http.StatusInternalServerError, "internal error")
		return
	}

	webutils.WriteJSON(w, http.StatusCreated, toRentalResponse(rental))
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
	return rentalResponse{
		ID:      rental.ID.String(),
		PlotID:  rental.PlotID.String(),
		CropID:  rental.CropID.String(),
		StartAt: rental.StartAt.Format(time.RFC3339),
		EndAt:   rental.EndAt.Format(time.RFC3339),
	}
}
