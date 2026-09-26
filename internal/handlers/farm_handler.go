package handlers

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/Neue-Konzepte-BaaS/backend/internal/middleware"
	"github.com/Neue-Konzepte-BaaS/backend/internal/models"
	"github.com/Neue-Konzepte-BaaS/backend/internal/services"
	"github.com/Neue-Konzepte-BaaS/backend/internal/webutils"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

type FarmHandler struct {
	farmService services.FarmService
}

func NewFarmHandler(farmService services.FarmService) *FarmHandler {
	return &FarmHandler{farmService: farmService}
}

type farmResponse struct {
	ID                string  `json:"id"`
	FarmerID          string  `json:"farmerId"`
	Name              string  `json:"name"`
	Address           string  `json:"address"`
	Description       string  `json:"description"`
	FoundedAt         *string `json:"foundedAt"`
	TotalSquareMeters float64 `json:"totalSquareMeters"`
}

// GetFarm returns public details of the farm with the given id, including
// the total area of plots it offers on the platform. It is a public endpoint
// (no auth required) — customers browsing plots use it.
func (h *FarmHandler) GetFarm(w http.ResponseWriter, r *http.Request) {
	farmID, err := uuid.Parse(chi.URLParam(r, "farmID"))
	if err != nil {
		webutils.WriteError(w, http.StatusBadRequest, "invalid farm id")
		return
	}

	farm, err := h.farmService.GetFarm(r.Context(), farmID)
	if errors.Is(err, services.ErrNotFound) {
		webutils.WriteError(w, http.StatusNotFound, "farm not found")
		return
	}
	if err != nil {
		slog.Error("getting farm failed", "error", err)
		webutils.WriteError(w, http.StatusInternalServerError, "internal error")
		return
	}

	var foundedAt *string
	if farm.FoundedAt != nil {
		s := farm.FoundedAt.Format(time.DateOnly)
		foundedAt = &s
	}

	webutils.WriteJSON(w, http.StatusOK, farmResponse{
		ID:                farm.ID.String(),
		FarmerID:          farm.FarmerID.String(),
		Name:              farm.Name,
		Address:           farm.Address,
		Description:       farm.Description,
		FoundedAt:         foundedAt,
		TotalSquareMeters: farm.TotalSquareMeters,
	})
}

type farmOwnerResponse struct {
	FirstName string `json:"firstName"`
	LastName  string `json:"lastName"`
	Email     string `json:"email"`
}

type farmListingResponse struct {
	ID         string            `json:"id"`
	FarmerID   string            `json:"farmerId"`
	Name       string            `json:"name"`
	Address    string            `json:"address"`
	PostalCode int32             `json:"postalCode"`
	Owner      farmOwnerResponse `json:"owner"`
	CreatedAt  string            `json:"createdAt"`
	// fields and plots are the shapes GET /api/statistics already returns, so
	// a client renders them with the code it already has.
	Fields        fieldStatisticsResponse `json:"fields"`
	Plots         plotStatisticsResponse  `json:"plots"`
	ActiveRentals int64                   `json:"activeRentals"`
}

type farmPageResponse struct {
	Items  []farmListingResponse `json:"items"`
	Total  int64                 `json:"total"`
	Limit  int32                 `json:"limit"`
	Offset int32                 `json:"offset"`
}

// ListFarms returns one page of every farm on the platform, with its owner and
// its holdings. Unlike GetFarm it is a back-office view, so it must be mounted
// behind RequireAuth and RequireRole(models.RoleAdmin).
func (h *FarmHandler) ListFarms(w http.ResponseWriter, r *http.Request) {
	claims := middleware.MustClaimsFromContext(r.Context())
	query := r.URL.Query()

	limit, err := parseLimit(query.Get("limit"), defaultPageLimit, maxPageLimit)
	if err != nil {
		webutils.WriteError(w, http.StatusBadRequest, "limit must be a number between 1 and 100")
		return
	}
	offset, err := parseOffset(query.Get("offset"))
	if err != nil {
		webutils.WriteError(w, http.StatusBadRequest, "offset must be a number of 0 or more")
		return
	}

	var postalCode *int32
	if raw := strings.TrimSpace(query.Get("postalCode")); raw != "" {
		parsed, err := strconv.ParseInt(raw, 10, 32)
		if err != nil {
			webutils.WriteError(w, http.StatusBadRequest, "postalCode must be a number")
			return
		}
		code := int32(parsed)
		postalCode = &code
	}

	page, err := h.farmService.ListFarms(r.Context(), claims.Role, models.FarmListFilter{
		Query:      query.Get("q"),
		PostalCode: postalCode,
		Limit:      limit,
		Offset:     offset,
	})
	if errors.Is(err, services.ErrForbidden) {
		webutils.WriteError(w, http.StatusForbidden, "insufficient permissions")
		return
	}
	if err != nil {
		slog.Error("listing farms failed", "error", err)
		webutils.WriteError(w, http.StatusInternalServerError, "internal error")
		return
	}

	webutils.WriteJSON(w, http.StatusOK, toFarmPageResponse(page))
}

func toFarmPageResponse(page models.Page[models.FarmListing]) farmPageResponse {
	items := make([]farmListingResponse, len(page.Items))
	for i, farm := range page.Items {
		items[i] = farmListingResponse{
			ID:         farm.ID.String(),
			FarmerID:   farm.FarmerID.String(),
			Name:       farm.Name,
			Address:    farm.Address,
			PostalCode: farm.PostalCode,
			Owner: farmOwnerResponse{
				FirstName: farm.FirstName,
				LastName:  farm.LastName,
				Email:     farm.Email,
			},
			CreatedAt: farm.CreatedAt.Format(time.RFC3339),
			Fields: fieldStatisticsResponse{
				Total:            farm.Fields.Total,
				AreaSquareMeters: farm.Fields.AreaSquareMeters,
			},
			Plots: plotStatisticsResponse{
				Total:            farm.Plots.Total,
				Rented:           farm.Plots.Rented,
				Available:        farm.Plots.Available,
				AreaSquareMeters: farm.Plots.AreaSquareMeters,
				OccupancyRate:    farm.Plots.OccupancyRate,
			},
			ActiveRentals: farm.ActiveRentals,
		}
	}
	return farmPageResponse{
		Items:  items,
		Total:  page.Total,
		Limit:  page.Limit,
		Offset: page.Offset,
	}
}

type farmCropRateResponse struct {
	CropID                  string `json:"cropId"`
	PriceCentsPerSqmPerWeek int32  `json:"priceCentsPerSqmPerWeek"`
}

type farmCropRatesResponse struct {
	Rates []farmCropRateResponse `json:"rates"`
}

type setFarmCropRatesRequest struct {
	Rates []struct {
		CropID                  string `json:"cropId"`
		PriceCentsPerSqmPerWeek int32  `json:"priceCentsPerSqmPerWeek"`
	} `json:"rates"`
}

func toFarmCropRatesResponse(rates []models.FarmCropRate) farmCropRatesResponse {
	res := make([]farmCropRateResponse, len(rates))
	for i, rate := range rates {
		res[i] = farmCropRateResponse{CropID: rate.Crop.String(), PriceCentsPerSqmPerWeek: rate.PriceCentsPerSqmPerWeek}
	}
	return farmCropRatesResponse{Rates: res}
}

// GetCropRates returns the authenticated farmer's own farm-wide crop rates.
// It must be mounted behind RequireAuth and RequireRole(models.RoleFarmer).
func (h *FarmHandler) GetCropRates(w http.ResponseWriter, r *http.Request) {
	claims := middleware.MustClaimsFromContext(r.Context())

	rates, err := h.farmService.GetCropRates(r.Context(), claims.UserID)
	if err != nil {
		slog.Error("getting farm crop rates failed", "error", err)
		webutils.WriteError(w, http.StatusInternalServerError, "internal error")
		return
	}

	webutils.WriteJSON(w, http.StatusOK, toFarmCropRatesResponse(rates))
}

// SetCropRates fully replaces the authenticated farmer's own farm-wide crop
// rates. It must be mounted behind RequireAuth and
// RequireRole(models.RoleFarmer).
func (h *FarmHandler) SetCropRates(w http.ResponseWriter, r *http.Request) {
	var req setFarmCropRatesRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		webutils.WriteError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	rates := make([]models.FarmCropRate, len(req.Rates))
	for i, rate := range req.Rates {
		cropID, err := uuid.Parse(rate.CropID)
		if err != nil {
			webutils.WriteError(w, http.StatusBadRequest, "invalid crop id")
			return
		}
		if rate.PriceCentsPerSqmPerWeek <= 0 {
			webutils.WriteError(w, http.StatusBadRequest, "priceCentsPerSqmPerWeek must be positive")
			return
		}
		rates[i] = models.FarmCropRate{Crop: cropID, PriceCentsPerSqmPerWeek: rate.PriceCentsPerSqmPerWeek}
	}

	claims := middleware.MustClaimsFromContext(r.Context())

	saved, err := h.farmService.SetCropRates(r.Context(), claims.UserID, rates)
	if errors.Is(err, services.ErrNotFound) {
		webutils.WriteError(w, http.StatusNotFound, "crop not found")
		return
	}
	if err != nil {
		slog.Error("setting farm crop rates failed", "error", err)
		webutils.WriteError(w, http.StatusInternalServerError, "internal error")
		return
	}

	webutils.WriteJSON(w, http.StatusOK, toFarmCropRatesResponse(saved))
}
