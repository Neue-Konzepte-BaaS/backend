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
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

type RipenessNoticeHandler struct {
	ripenessNoticeService services.RipenessNoticeService
}

func NewRipenessNoticeHandler(ripenessNoticeService services.RipenessNoticeService) *RipenessNoticeHandler {
	return &RipenessNoticeHandler{ripenessNoticeService: ripenessNoticeService}
}

type createRipenessNoticeRequest struct {
	CropID string `json:"crop_id"`
}

type ripenessNoticeResponse struct {
	ID        string    `json:"id"`
	Farmer    string    `json:"farmer"`
	FarmName  string    `json:"farm_name"`
	Field     string    `json:"field"`
	FieldName string    `json:"field_name"`
	Crop      string    `json:"crop"`
	CropName  string    `json:"crop_name"`
	CreatedAt time.Time `json:"created_at"`
}

type createRipenessNoticeResponse struct {
	ripenessNoticeResponse
	Recipients int `json:"recipients"`
}

func toRipenessNoticeResponse(notice models.RipenessNoticeWithDetails) ripenessNoticeResponse {
	return ripenessNoticeResponse{
		ID:        notice.ID.String(),
		Farmer:    notice.Farmer.String(),
		FarmName:  notice.FarmName,
		Field:     notice.Field.String(),
		FieldName: notice.FieldName,
		Crop:      notice.Crop.String(),
		CropName:  notice.CropName,
		CreatedAt: notice.CreatedAt,
	}
}

// Create posts a ripeness notice for a field owned by the authenticated
// farmer and mails everyone currently renting a plot of it with that crop.
// It must be mounted behind RequireAuth and RequireRole(models.RoleFarmer).
func (h *RipenessNoticeHandler) Create(w http.ResponseWriter, r *http.Request) {
	fieldID, err := uuid.Parse(chi.URLParam(r, "fieldID"))
	if err != nil {
		webutils.WriteError(w, http.StatusBadRequest, "invalid field id")
		return
	}

	var req createRipenessNoticeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		webutils.WriteError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	cropID, err := uuid.Parse(req.CropID)
	if err != nil {
		webutils.WriteError(w, http.StatusBadRequest, "crop_id is required and must be a valid id")
		return
	}

	claims := middleware.MustClaimsFromContext(r.Context())

	notice, recipients, err := h.ripenessNoticeService.CreateRipenessNotice(r.Context(), claims.UserID, fieldID, cropID)
	if errors.Is(err, services.ErrNotFound) {
		webutils.WriteError(w, http.StatusNotFound, "field or crop not found")
		return
	}
	if errors.Is(err, services.ErrForbidden) {
		webutils.WriteError(w, http.StatusForbidden, "field is not owned by this farmer")
		return
	}
	if err != nil {
		slog.Error("creating ripeness notice failed", "error", err)
		webutils.WriteError(w, http.StatusInternalServerError, "internal error")
		return
	}

	webutils.WriteJSON(w, http.StatusCreated, createRipenessNoticeResponse{
		ripenessNoticeResponse: toRipenessNoticeResponse(notice),
		Recipients:             recipients,
	})
}
