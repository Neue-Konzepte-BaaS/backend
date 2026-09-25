package handlers

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

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

	webutils.WriteJSON(w, http.StatusOK, toFarmResponse(farm))
}

func toFarmResponse(farm models.Farm) farmResponse {
	var foundedAt *string
	if farm.FoundedAt != nil {
		s := farm.FoundedAt.Format(time.DateOnly)
		foundedAt = &s
	}

	return farmResponse{
		ID:                farm.ID.String(),
		FarmerID:          farm.FarmerID.String(),
		Name:              farm.Name,
		Address:           farm.Address,
		Description:       farm.Description,
		FoundedAt:         foundedAt,
		TotalSquareMeters: farm.TotalSquareMeters,
	}
}

// Bounds for what a farmer may write about their farm. The name and address
// are shown on every plot card and the farm page, the description only on the
// farm page.
const (
	maxFarmName        = 100
	maxFarmAddress     = 200
	maxFarmDescription = 2_000
)

// updateFarmRequest is a full replacement of the editable fields: an absent
// or null foundedAt clears the date, like an empty description clears that.
type updateFarmRequest struct {
	Name        string  `json:"name"`
	Address     string  `json:"address"`
	Description string  `json:"description"`
	FoundedAt   *string `json:"foundedAt"`
}

// GetMyFarm returns the calling farmer's own farm, in the same shape as
// GetFarm. It must be mounted behind RequireAuth and
// RequireRole(models.RoleFarmer).
func (h *FarmHandler) GetMyFarm(w http.ResponseWriter, r *http.Request) {
	claims := middleware.MustClaimsFromContext(r.Context())

	farm, err := h.farmService.GetMyFarm(r.Context(), claims.UserID)
	if errors.Is(err, services.ErrNotFound) {
		webutils.WriteError(w, http.StatusNotFound, "farm not found")
		return
	}
	if err != nil {
		slog.Error("getting own farm failed", "error", err)
		webutils.WriteError(w, http.StatusInternalServerError, "internal error")
		return
	}

	webutils.WriteJSON(w, http.StatusOK, toFarmResponse(farm))
}

// UpdateMyFarm overwrites the calling farmer's farm details. It must be
// mounted behind RequireAuth and RequireRole(models.RoleFarmer).
func (h *FarmHandler) UpdateMyFarm(w http.ResponseWriter, r *http.Request) {
	claims := middleware.MustClaimsFromContext(r.Context())

	var req updateFarmRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		webutils.WriteError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	update := models.FarmUpdate{
		Name:        strings.TrimSpace(req.Name),
		Address:     strings.TrimSpace(req.Address),
		Description: strings.TrimSpace(req.Description),
	}
	switch {
	case update.Name == "":
		webutils.WriteError(w, http.StatusBadRequest, "name is required")
		return
	case update.Address == "":
		webutils.WriteError(w, http.StatusBadRequest, "address is required")
		return
	case utf8.RuneCountInString(update.Name) > maxFarmName:
		webutils.WriteError(w, http.StatusBadRequest, "name is too long")
		return
	case utf8.RuneCountInString(update.Address) > maxFarmAddress:
		webutils.WriteError(w, http.StatusBadRequest, "address is too long")
		return
	case utf8.RuneCountInString(update.Description) > maxFarmDescription:
		webutils.WriteError(w, http.StatusBadRequest, "description is too long")
		return
	}

	if req.FoundedAt != nil && strings.TrimSpace(*req.FoundedAt) != "" {
		founded, err := time.Parse(time.DateOnly, strings.TrimSpace(*req.FoundedAt))
		if err != nil {
			webutils.WriteError(w, http.StatusBadRequest, "foundedAt must be a date (YYYY-MM-DD)")
			return
		}
		// Compared as calendar dates in UTC, so "today" is always allowed
		// whatever the server's zone.
		if founded.After(time.Now().UTC().Truncate(24 * time.Hour)) {
			webutils.WriteError(w, http.StatusBadRequest, "foundedAt must not be in the future")
			return
		}
		update.FoundedAt = &founded
	}

	farm, err := h.farmService.UpdateMyFarm(r.Context(), claims.UserID, update)
	if errors.Is(err, services.ErrNotFound) {
		webutils.WriteError(w, http.StatusNotFound, "farm not found")
		return
	}
	if err != nil {
		slog.Error("updating own farm failed", "error", err)
		webutils.WriteError(w, http.StatusInternalServerError, "internal error")
		return
	}

	webutils.WriteJSON(w, http.StatusOK, toFarmResponse(farm))
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
