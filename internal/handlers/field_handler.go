package handlers

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strings"

	"github.com/Neue-Konzepte-BaaS/backend/internal/middleware"
	"github.com/Neue-Konzepte-BaaS/backend/internal/services"
	"github.com/Neue-Konzepte-BaaS/backend/internal/webutils"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	geom "github.com/twpayne/go-geom"
	"github.com/twpayne/go-geom/encoding/geojson"
)

type FieldHandler struct {
	fieldService services.FieldService
	plotService  services.PlotService
}

func NewFieldHandler(fieldService services.FieldService, plotService services.PlotService) *FieldHandler {
	return &FieldHandler{fieldService: fieldService, plotService: plotService}
}

type createFieldRequest struct {
	Name        string          `json:"name"`
	Coordinates json.RawMessage `json:"coordinates"`
}

type createPlotRequest struct {
	Name        string          `json:"name"`
	Coordinates json.RawMessage `json:"coordinates"`
}

type fieldResponse struct {
	ID          string          `json:"id"`
	Name        string          `json:"name"`
	Farmer      string          `json:"farmer"`
	Coordinates json.RawMessage `json:"coordinates"`
}

type plotResponse struct {
	ID          string          `json:"id"`
	Name        string          `json:"name"`
	Field       string          `json:"field"`
	Coordinates json.RawMessage `json:"coordinates"`
}

type fieldWithPlotsResponse struct {
	ID          string          `json:"id"`
	Name        string          `json:"name"`
	Farmer      string          `json:"farmer"`
	Coordinates json.RawMessage `json:"coordinates"`
	Plots       []plotResponse  `json:"plots"`
	Crops       []cropResponse  `json:"crops"`
}

// decodePolygon parses a GeoJSON Polygon geometry, e.g.
// {"type":"Polygon","coordinates":[[[lon,lat],...]]}.
func decodePolygon(raw json.RawMessage) (*geom.Polygon, error) {
	var g geom.T
	if err := geojson.Unmarshal(raw, &g); err != nil {
		return nil, err
	}
	polygon, ok := g.(*geom.Polygon)
	if !ok {
		return nil, errors.New("coordinates must be a GeoJSON Polygon")
	}
	return polygon, nil
}

func encodePolygon(polygon *geom.Polygon) json.RawMessage {
	raw, err := geojson.Marshal(polygon)
	if err != nil {
		slog.Error("encoding polygon", "error", err)
		return nil
	}
	return raw
}

// CreateField creates a field owned by the authenticated farmer. It must be
// mounted behind RequireAuth and RequireRole(models.RoleFarmer).
func (h *FieldHandler) CreateField(w http.ResponseWriter, r *http.Request) {
	var req createFieldRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		webutils.WriteError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	req.Name = strings.TrimSpace(req.Name)
	if req.Name == "" {
		webutils.WriteError(w, http.StatusBadRequest, "name is required")
		return
	}

	coordinates, err := decodePolygon(req.Coordinates)
	if err != nil {
		webutils.WriteError(w, http.StatusBadRequest, "coordinates must be a valid GeoJSON Polygon")
		return
	}

	claims := middleware.MustClaimsFromContext(r.Context())

	field, err := h.fieldService.CreateField(r.Context(), claims.UserID, req.Name, coordinates)
	if errors.Is(err, services.ErrInvalidGeometry) {
		webutils.WriteError(w, http.StatusBadRequest, "field must be a rectangle")
		return
	}
	if err != nil {
		slog.Error("creating field failed", "error", err)
		webutils.WriteError(w, http.StatusInternalServerError, "internal error")
		return
	}

	webutils.WriteJSON(w, http.StatusCreated, fieldResponse{
		ID:          field.ID.String(),
		Name:        field.Name,
		Farmer:      field.Farmer.String(),
		Coordinates: encodePolygon(field.Coordinates),
	})
}

// GetFields returns all fields owned by the authenticated farmer, along with
// their plots. It must be mounted behind RequireAuth and
// RequireRole(models.RoleFarmer).
func (h *FieldHandler) GetFields(w http.ResponseWriter, r *http.Request) {
	claims := middleware.MustClaimsFromContext(r.Context())

	fields, err := h.fieldService.GetFieldsWithPlots(r.Context(), claims.UserID)
	if err != nil {
		slog.Error("getting fields failed", "error", err)
		webutils.WriteError(w, http.StatusInternalServerError, "internal error")
		return
	}

	res := make([]fieldWithPlotsResponse, len(fields))
	for i, field := range fields {
		plots := make([]plotResponse, len(field.Plots))
		for j, plot := range field.Plots {
			plots[j] = plotResponse{
				ID:          plot.ID.String(),
				Name:        plot.Name,
				Field:       plot.Field.String(),
				Coordinates: encodePolygon(plot.Coordinates),
			}
		}
		res[i] = fieldWithPlotsResponse{
			ID:          field.ID.String(),
			Name:        field.Name,
			Farmer:      field.Farmer.String(),
			Coordinates: encodePolygon(field.Coordinates),
			Plots:       plots,
			Crops:       toCropResponses(field.Crops),
		}
	}

	webutils.WriteJSON(w, http.StatusOK, res)
}

// CreatePlot creates a plot on a field owned by the authenticated farmer. It
// must be mounted behind RequireAuth and RequireRole(models.RoleFarmer).
func (h *FieldHandler) CreatePlot(w http.ResponseWriter, r *http.Request) {
	fieldID, err := uuid.Parse(chi.URLParam(r, "fieldID"))
	if err != nil {
		webutils.WriteError(w, http.StatusBadRequest, "invalid field id")
		return
	}

	var req createPlotRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		webutils.WriteError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	req.Name = strings.TrimSpace(req.Name)
	if req.Name == "" {
		webutils.WriteError(w, http.StatusBadRequest, "name is required")
		return
	}

	coordinates, err := decodePolygon(req.Coordinates)
	if err != nil {
		webutils.WriteError(w, http.StatusBadRequest, "coordinates must be a valid GeoJSON Polygon")
		return
	}

	claims := middleware.MustClaimsFromContext(r.Context())

	plot, err := h.plotService.CreatePlot(r.Context(), claims.UserID, fieldID, req.Name, coordinates)
	if errors.Is(err, services.ErrNotFound) {
		webutils.WriteError(w, http.StatusNotFound, "field not found")
		return
	}
	if errors.Is(err, services.ErrForbidden) {
		webutils.WriteError(w, http.StatusForbidden, "field is not owned by this farmer")
		return
	}
	if errors.Is(err, services.ErrInvalidGeometry) {
		webutils.WriteError(w, http.StatusBadRequest, "plot must be within the field's boundaries")
		return
	}
	if err != nil {
		slog.Error("creating plot failed", "error", err)
		webutils.WriteError(w, http.StatusInternalServerError, "internal error")
		return
	}

	webutils.WriteJSON(w, http.StatusCreated, plotResponse{
		ID:          plot.ID.String(),
		Name:        plot.Name,
		Field:       plot.Field.String(),
		Coordinates: encodePolygon(plot.Coordinates),
	})
}
