package handlers

import (
	"errors"
	"log/slog"
	"net/http"
	"time"

	"github.com/Neue-Konzepte-BaaS/backend/internal/middleware"
	"github.com/Neue-Konzepte-BaaS/backend/internal/models"
	"github.com/Neue-Konzepte-BaaS/backend/internal/services"
	"github.com/Neue-Konzepte-BaaS/backend/internal/webutils"
)

type StatisticsHandler struct {
	statisticsService services.StatisticsService
}

func NewStatisticsHandler(statisticsService services.StatisticsService) *StatisticsHandler {
	return &StatisticsHandler{statisticsService: statisticsService}
}

type fieldStatisticsResponse struct {
	Total            int64   `json:"total"`
	AreaSquareMeters float64 `json:"areaSquareMeters"`
}

type plotStatisticsResponse struct {
	Total            int64   `json:"total"`
	Rented           int64   `json:"rented"`
	Available        int64   `json:"available"`
	AreaSquareMeters float64 `json:"areaSquareMeters"`
	OccupancyRate    float64 `json:"occupancyRate"`
}

type rentalStatisticsResponse struct {
	Total      int64 `json:"total"`
	Active     int64 `json:"active"`
	Last30Days int64 `json:"last30Days"`
}

type accountStatisticsResponse struct {
	Total                int64 `json:"total"`
	Farmers              int64 `json:"farmers"`
	Customers            int64 `json:"customers"`
	RegisteredLast30Days int64 `json:"registeredLast30Days"`
}

type statisticsResponse struct {
	Scope       string                     `json:"scope"`
	GeneratedAt string                     `json:"generatedAt"`
	Fields      fieldStatisticsResponse    `json:"fields"`
	Plots       plotStatisticsResponse     `json:"plots"`
	Rentals     rentalStatisticsResponse   `json:"rentals"`
	Accounts    *accountStatisticsResponse `json:"accounts,omitempty"`
}

// GetStatistics returns the statistics the authenticated account may see: a
// farmer gets their own farm, an admin gets the whole platform. It must be
// mounted behind RequireAuth and
// RequireAnyRole(models.RoleFarmer, models.RoleAdmin).
func (h *StatisticsHandler) GetStatistics(w http.ResponseWriter, r *http.Request) {
	claims := middleware.MustClaimsFromContext(r.Context())

	stats, err := h.statisticsService.GetStatistics(r.Context(), claims.UserID, claims.Role)
	if errors.Is(err, services.ErrForbidden) {
		webutils.WriteError(w, http.StatusForbidden, "insufficient permissions")
		return
	}
	if err != nil {
		slog.Error("getting statistics failed", "error", err)
		webutils.WriteError(w, http.StatusInternalServerError, "internal error")
		return
	}

	webutils.WriteJSON(w, http.StatusOK, toStatisticsResponse(stats))
}

func toStatisticsResponse(stats models.Statistics) statisticsResponse {
	res := statisticsResponse{
		Scope:       string(stats.Scope),
		GeneratedAt: stats.GeneratedAt.Format(time.RFC3339),
		Fields: fieldStatisticsResponse{
			Total:            stats.Fields.Total,
			AreaSquareMeters: stats.Fields.AreaSquareMeters,
		},
		Plots: plotStatisticsResponse{
			Total:            stats.Plots.Total,
			Rented:           stats.Plots.Rented,
			Available:        stats.Plots.Available,
			AreaSquareMeters: stats.Plots.AreaSquareMeters,
			OccupancyRate:    stats.Plots.OccupancyRate,
		},
		Rentals: rentalStatisticsResponse{
			Total:      stats.Rentals.Total,
			Active:     stats.Rentals.Active,
			Last30Days: stats.Rentals.Last30Days,
		},
	}
	if stats.Accounts != nil {
		res.Accounts = &accountStatisticsResponse{
			Total:                stats.Accounts.Total,
			Farmers:              stats.Accounts.Farmers,
			Customers:            stats.Accounts.Customers,
			RegisteredLast30Days: stats.Accounts.RegisteredLast30Days,
		}
	}
	return res
}
