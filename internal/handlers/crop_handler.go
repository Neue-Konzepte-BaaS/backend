package handlers

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strings"

	"github.com/Neue-Konzepte-BaaS/backend/internal/middleware"
	"github.com/Neue-Konzepte-BaaS/backend/internal/models"
	"github.com/Neue-Konzepte-BaaS/backend/internal/services"
	"github.com/Neue-Konzepte-BaaS/backend/internal/webutils"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

type CropHandler struct {
	cropService services.CropService
}

func NewCropHandler(cropService services.CropService) *CropHandler {
	return &CropHandler{cropService: cropService}
}

type cropResponse struct {
	ID             string `json:"id"`
	Name           string `json:"name"`
	DurationMonths int32  `json:"durationMonths"`
}

type setPlotCropsRequest struct {
	CropIDs []string `json:"cropIds"`
}

type createCropRequest struct {
	Name           string `json:"name"`
	DurationMonths int32  `json:"durationMonths"`
}

func toCropResponse(crop models.Crop) cropResponse {
	return cropResponse{
		ID:             crop.ID.String(),
		Name:           crop.Name,
		DurationMonths: crop.DurationMonths,
	}
}

func toCropResponses(crops []models.Crop) []cropResponse {
	res := make([]cropResponse, len(crops))
	for i, crop := range crops {
		res[i] = toCropResponse(crop)
	}
	return res
}

// CreateCrop adds a new crop to the catalog. It must be mounted behind
// RequireAuth and RequireRole(models.RoleAdmin).
func (h *CropHandler) CreateCrop(w http.ResponseWriter, r *http.Request) {
	var req createCropRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		webutils.WriteError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	req.Name = strings.TrimSpace(req.Name)
	if req.Name == "" {
		webutils.WriteError(w, http.StatusBadRequest, "name is required")
		return
	}
	if req.DurationMonths <= 0 {
		webutils.WriteError(w, http.StatusBadRequest, "durationMonths must be positive")
		return
	}

	crop, err := h.cropService.CreateCrop(r.Context(), req.Name, req.DurationMonths)
	if errors.Is(err, services.ErrCropNameTaken) {
		webutils.WriteError(w, http.StatusConflict, "crop name already exists")
		return
	}
	if err != nil {
		slog.Error("creating crop failed", "error", err)
		webutils.WriteError(w, http.StatusInternalServerError, "internal error")
		return
	}

	webutils.WriteJSON(w, http.StatusCreated, toCropResponse(crop))
}

// DeleteCrop removes a crop from the catalog. It must be mounted behind
// RequireAuth and RequireRole(models.RoleAdmin).
func (h *CropHandler) DeleteCrop(w http.ResponseWriter, r *http.Request) {
	cropID, err := uuid.Parse(chi.URLParam(r, "cropID"))
	if err != nil {
		webutils.WriteError(w, http.StatusBadRequest, "invalid crop id")
		return
	}

	if err := h.cropService.DeleteCrop(r.Context(), cropID); errors.Is(err, services.ErrConflict) {
		webutils.WriteError(w, http.StatusConflict, "crop is referenced by a rental")
		return
	} else if err != nil {
		slog.Error("deleting crop failed", "error", err)
		webutils.WriteError(w, http.StatusInternalServerError, "internal error")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// GetAllCrops returns the full crop catalog.
func (h *CropHandler) GetAllCrops(w http.ResponseWriter, r *http.Request) {
	crops, err := h.cropService.GetAllCrops(r.Context())
	if err != nil {
		slog.Error("getting crops failed", "error", err)
		webutils.WriteError(w, http.StatusInternalServerError, "internal error")
		return
	}

	webutils.WriteJSON(w, http.StatusOK, toCropResponses(crops))
}

// SetPlotCrops replaces the crops offered by a plot on a field owned by the
// authenticated farmer. It must be mounted behind RequireAuth and
// RequireRole(models.RoleFarmer).
func (h *CropHandler) SetPlotCrops(w http.ResponseWriter, r *http.Request) {
	plotID, err := uuid.Parse(chi.URLParam(r, "plotID"))
	if err != nil {
		webutils.WriteError(w, http.StatusBadRequest, "invalid plot id")
		return
	}

	var req setPlotCropsRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		webutils.WriteError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	cropIDs := make([]uuid.UUID, len(req.CropIDs))
	for i, raw := range req.CropIDs {
		cropID, err := uuid.Parse(raw)
		if err != nil {
			webutils.WriteError(w, http.StatusBadRequest, "invalid crop id")
			return
		}
		cropIDs[i] = cropID
	}

	claims := middleware.MustClaimsFromContext(r.Context())

	crops, err := h.cropService.SetPlotCrops(r.Context(), claims.UserID, plotID, cropIDs)
	if errors.Is(err, services.ErrNotFound) {
		webutils.WriteError(w, http.StatusNotFound, "plot or crop not found")
		return
	}
	if errors.Is(err, services.ErrForbidden) {
		webutils.WriteError(w, http.StatusForbidden, "field is not owned by this farmer")
		return
	}
	if err != nil {
		slog.Error("setting plot crops failed", "error", err)
		webutils.WriteError(w, http.StatusInternalServerError, "internal error")
		return
	}

	webutils.WriteJSON(w, http.StatusOK, toCropResponses(crops))
}
