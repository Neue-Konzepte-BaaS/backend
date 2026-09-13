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

type setFieldCropsRequest struct {
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

// SetFieldCrops replaces the crops offered by a field owned by the
// authenticated farmer. It must be mounted behind RequireAuth and
// RequireRole(models.RoleFarmer).
func (h *CropHandler) SetFieldCrops(w http.ResponseWriter, r *http.Request) {
	fieldID, err := uuid.Parse(chi.URLParam(r, "fieldID"))
	if err != nil {
		webutils.WriteError(w, http.StatusBadRequest, "invalid field id")
		return
	}

	var req setFieldCropsRequest
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

	crops, err := h.cropService.SetFieldCrops(r.Context(), claims.UserID, fieldID, cropIDs)
	if errors.Is(err, services.ErrNotFound) {
		webutils.WriteError(w, http.StatusNotFound, "field or crop not found")
		return
	}
	if errors.Is(err, services.ErrForbidden) {
		webutils.WriteError(w, http.StatusForbidden, "field is not owned by this farmer")
		return
	}
	if err != nil {
		slog.Error("setting field crops failed", "error", err)
		webutils.WriteError(w, http.StatusInternalServerError, "internal error")
		return
	}

	webutils.WriteJSON(w, http.StatusOK, toCropResponses(crops))
}
