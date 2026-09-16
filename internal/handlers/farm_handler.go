package handlers

import (
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
)

type FarmHandler struct {
	farmService services.FarmService
}

func NewFarmHandler(farmService services.FarmService) *FarmHandler {
	return &FarmHandler{farmService: farmService}
}

type farmOwnerResponse struct {
	FirstName string `json:"firstName"`
	LastName  string `json:"lastName"`
	Email     string `json:"email"`
}

type farmListingResponse struct {
	AccountID  string            `json:"accountId"`
	FarmName   string            `json:"farmName"`
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

// ListFarms returns one page of every farm on the platform. It must be mounted
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
			AccountID:  farm.Account.String(),
			FarmName:   farm.FarmName,
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
