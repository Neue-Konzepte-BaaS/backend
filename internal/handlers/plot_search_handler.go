package handlers

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strconv"
	"strings"

	"github.com/Neue-Konzepte-BaaS/backend/internal/models"
	"github.com/Neue-Konzepte-BaaS/backend/internal/services"
	"github.com/Neue-Konzepte-BaaS/backend/internal/webutils"
	"github.com/google/uuid"
)

type PlotSearchHandler struct {
	plotSearchService services.PlotSearchService
}

func NewPlotSearchHandler(plotSearchService services.PlotSearchService) *PlotSearchHandler {
	return &PlotSearchHandler{plotSearchService: plotSearchService}
}

type plotCropOfferingResponse struct {
	cropResponse
	PriceCents int32 `json:"priceCents"`
}

type nearbyPlotResponse struct {
	ID               string                     `json:"id"`
	Name             string                     `json:"name"`
	Field            string                     `json:"field"`
	Farm             string                     `json:"farm"`
	Coordinates      json.RawMessage            `json:"coordinates"`
	AreaSquareMeters float64                    `json:"areaSquareMeters"`
	DistanceMeters   float64                    `json:"distanceMeters"`
	Crops            []plotCropOfferingResponse `json:"crops"`
}

func toPlotCropOfferingResponses(offerings []models.PlotCropOffering) []plotCropOfferingResponse {
	res := make([]plotCropOfferingResponse, len(offerings))
	for i, offering := range offerings {
		res[i] = plotCropOfferingResponse{
			cropResponse: toCropResponse(offering.Crop),
			PriceCents:   offering.PriceCents,
		}
	}
	return res
}

// FindNearestPlots returns the plots nearest to a search point, given
// either lat/lon query params or a postalCode/city query param. It is a
// public endpoint (no auth required).
func (h *PlotSearchHandler) FindNearestPlots(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()

	limit, err := parseLimit(query.Get("limit"), defaultPageLimit, maxPageLimit)
	if err != nil {
		webutils.WriteError(w, http.StatusBadRequest, "limit must be a number between 1 and 100")
		return
	}

	latStr := query.Get("lat")
	lonStr := query.Get("lon")
	postalCode := strings.TrimSpace(query.Get("postalCode"))
	city := strings.TrimSpace(query.Get("city"))

	var farm *uuid.UUID
	if farmStr := strings.TrimSpace(query.Get("farm")); farmStr != "" {
		id, parseErr := uuid.Parse(farmStr)
		if parseErr != nil {
			webutils.WriteError(w, http.StatusBadRequest, "farm must be a valid id")
			return
		}
		farm = &id
	}

	var plots []models.NearbyPlot
	switch {
	case latStr != "" && lonStr != "":
		lat, latErr := strconv.ParseFloat(latStr, 64)
		lon, lonErr := strconv.ParseFloat(lonStr, 64)
		if latErr != nil || lonErr != nil {
			webutils.WriteError(w, http.StatusBadRequest, "lat and lon must be numbers")
			return
		}
		plots, err = h.plotSearchService.FindNearestByCoordinates(r.Context(), lon, lat, farm, limit)
	case postalCode != "" || city != "":
		plots, err = h.plotSearchService.FindNearestByLocation(r.Context(), postalCode, city, farm, limit)
	default:
		webutils.WriteError(w, http.StatusBadRequest, "provide lat and lon, or postalCode or city")
		return
	}

	if errors.Is(err, services.ErrNotFound) {
		webutils.WriteError(w, http.StatusNotFound, "no location found for the given postal code or city")
		return
	}
	if err != nil {
		slog.Error("finding nearest plots failed", "error", err)
		webutils.WriteError(w, http.StatusInternalServerError, "internal error")
		return
	}

	res := make([]nearbyPlotResponse, len(plots))
	for i, plot := range plots {
		res[i] = nearbyPlotResponse{
			ID:               plot.ID.String(),
			Name:             plot.Name,
			Field:            plot.Field.String(),
			Farm:             plot.Farm.String(),
			Coordinates:      encodePolygon(plot.Coordinates),
			AreaSquareMeters: plot.AreaSquareMeters,
			DistanceMeters:   plot.DistanceMeters,
			Crops:            toPlotCropOfferingResponses(plot.Crops),
		}
	}

	webutils.WriteJSON(w, http.StatusOK, res)
}
