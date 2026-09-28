package handlers

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/Neue-Konzepte-BaaS/backend/internal/middleware"
	"github.com/Neue-Konzepte-BaaS/backend/internal/models"
	"github.com/Neue-Konzepte-BaaS/backend/internal/services"
	"github.com/Neue-Konzepte-BaaS/backend/internal/webutils"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

type SeasonHandler struct {
	seasonService services.SeasonService
}

func NewSeasonHandler(seasonService services.SeasonService) *SeasonHandler {
	return &SeasonHandler{seasonService: seasonService}
}

type seasonRequest struct {
	Name       string `json:"name"`
	StartMonth int32  `json:"startMonth"`
	StartDay   int32  `json:"startDay"`
	EndMonth   int32  `json:"endMonth"`
	EndDay     int32  `json:"endDay"`
}

type seasonResponse struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	// FarmID is null for one of the global defaults, and the farm's id for
	// that farm's own version.
	FarmID     *string   `json:"farmId"`
	StartMonth int32     `json:"startMonth"`
	StartDay   int32     `json:"startDay"`
	EndMonth   int32     `json:"endMonth"`
	EndDay     int32     `json:"endDay"`
	CreatedAt  time.Time `json:"createdAt"`
	UpdatedAt  time.Time `json:"updatedAt"`
}

type assignCropSeasonRequest struct {
	SeasonID string `json:"seasonId"`
}

type cropSeasonRuleResponse struct {
	ID       string `json:"id"`
	CropID   string `json:"cropId"`
	SeasonID string `json:"seasonId"`
	// FarmID is null for the default rule, and the farm's id for that farm's
	// own rule.
	FarmID *string `json:"farmId"`
}

// cropSeasonResponse is one crop paired with the season it is effectively
// checked against for the caller - the default rule for an admin, or the
// effective rule (their own farm's if it has one, the default otherwise)
// for a farmer. Null season means the crop is unrestricted for them.
type cropSeasonResponse struct {
	cropResponse
	Season *seasonResponse `json:"season"`
}

func toSeasonResponse(season models.Season) seasonResponse {
	var farmID *string
	if season.Farm != nil {
		id := season.Farm.String()
		farmID = &id
	}
	return seasonResponse{
		ID:         season.ID.String(),
		Name:       season.Name,
		FarmID:     farmID,
		StartMonth: season.StartMonth,
		StartDay:   season.StartDay,
		EndMonth:   season.EndMonth,
		EndDay:     season.EndDay,
		CreatedAt:  season.CreatedAt,
		UpdatedAt:  season.UpdatedAt,
	}
}

func toSeasonResponses(seasons []models.Season) []seasonResponse {
	res := make([]seasonResponse, len(seasons))
	for i, season := range seasons {
		res[i] = toSeasonResponse(season)
	}
	return res
}

func toCropSeasonRuleResponse(rule models.CropSeasonRule) cropSeasonRuleResponse {
	var farmID *string
	if rule.Farm != nil {
		id := rule.Farm.String()
		farmID = &id
	}
	return cropSeasonRuleResponse{
		ID:       rule.ID.String(),
		CropID:   rule.Crop.String(),
		SeasonID: rule.Season.String(),
		FarmID:   farmID,
	}
}

// seasonEditor is the caller as the season service sees an editor.
func seasonEditor(r *http.Request) services.SeasonEditor {
	claims := middleware.MustClaimsFromContext(r.Context())
	return services.SeasonEditor{AccountID: claims.UserID, Role: claims.Role}
}

// decodeSeason reads and validates the body shared by create and update. It
// writes the error response itself and reports whether the caller should
// carry on.
func decodeSeason(w http.ResponseWriter, r *http.Request) (seasonRequest, bool) {
	var req seasonRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		webutils.WriteError(w, http.StatusBadRequest, "invalid request body")
		return req, false
	}

	req.Name = strings.TrimSpace(req.Name)
	if req.Name == "" {
		webutils.WriteError(w, http.StatusBadRequest, "name is required")
		return req, false
	}
	if req.StartMonth < 1 || req.StartMonth > 12 || req.EndMonth < 1 || req.EndMonth > 12 {
		webutils.WriteError(w, http.StatusBadRequest, "startMonth and endMonth must be between 1 and 12")
		return req, false
	}
	if req.StartDay < 1 || req.StartDay > 31 || req.EndDay < 1 || req.EndDay > 31 {
		webutils.WriteError(w, http.StatusBadRequest, "startDay and endDay must be between 1 and 31")
		return req, false
	}
	return req, true
}

// CreateSeason adds a season to the default set (admin) or the caller's own
// farm's set (farmer). It must be mounted behind RequireAuth and
// RequireAnyRole(models.RoleAdmin, models.RoleFarmer).
func (h *SeasonHandler) CreateSeason(w http.ResponseWriter, r *http.Request) {
	req, ok := decodeSeason(w, r)
	if !ok {
		return
	}

	season, err := h.seasonService.CreateSeason(r.Context(), seasonEditor(r), req.Name, req.StartMonth, req.StartDay, req.EndMonth, req.EndDay)
	if errors.Is(err, services.ErrForbidden) {
		webutils.WriteError(w, http.StatusForbidden, "no farm to write a season for")
		return
	}
	if err != nil {
		slog.Error("creating season failed", "error", err)
		webutils.WriteError(w, http.StatusInternalServerError, "internal error")
		return
	}

	webutils.WriteJSON(w, http.StatusCreated, toSeasonResponse(season))
}

// UpdateSeason rewrites one season. An admin may edit any default season, a
// farmer only their own farm's. It must be mounted behind RequireAuth and
// RequireAnyRole(models.RoleAdmin, models.RoleFarmer).
func (h *SeasonHandler) UpdateSeason(w http.ResponseWriter, r *http.Request) {
	seasonID, err := uuid.Parse(chi.URLParam(r, "seasonID"))
	if err != nil {
		webutils.WriteError(w, http.StatusBadRequest, "invalid season id")
		return
	}

	req, ok := decodeSeason(w, r)
	if !ok {
		return
	}

	season, err := h.seasonService.UpdateSeason(r.Context(), seasonEditor(r), seasonID, req.Name, req.StartMonth, req.StartDay, req.EndMonth, req.EndDay)
	if errors.Is(err, services.ErrNotFound) {
		webutils.WriteError(w, http.StatusNotFound, "season not found")
		return
	}
	if errors.Is(err, services.ErrForbidden) {
		webutils.WriteError(w, http.StatusForbidden, "no farm to write a season for")
		return
	}
	if err != nil {
		slog.Error("updating season failed", "error", err)
		webutils.WriteError(w, http.StatusInternalServerError, "internal error")
		return
	}

	webutils.WriteJSON(w, http.StatusOK, toSeasonResponse(season))
}

// DeleteSeason removes one season, under the same rules as UpdateSeason. It
// must be mounted behind RequireAuth and RequireAnyRole(models.RoleAdmin,
// models.RoleFarmer).
func (h *SeasonHandler) DeleteSeason(w http.ResponseWriter, r *http.Request) {
	seasonID, err := uuid.Parse(chi.URLParam(r, "seasonID"))
	if err != nil {
		webutils.WriteError(w, http.StatusBadRequest, "invalid season id")
		return
	}

	err = h.seasonService.DeleteSeason(r.Context(), seasonEditor(r), seasonID)
	if errors.Is(err, services.ErrNotFound) {
		webutils.WriteError(w, http.StatusNotFound, "season not found")
		return
	}
	if errors.Is(err, services.ErrForbidden) {
		webutils.WriteError(w, http.StatusForbidden, "no farm to write a season for")
		return
	}
	if err != nil {
		slog.Error("deleting season failed", "error", err)
		webutils.WriteError(w, http.StatusInternalServerError, "internal error")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// GetSeasons returns the default set (admin) or the union of the default set
// and the caller's own farm's seasons (farmer). It must be mounted behind
// RequireAuth and RequireAnyRole(models.RoleAdmin, models.RoleFarmer).
func (h *SeasonHandler) GetSeasons(w http.ResponseWriter, r *http.Request) {
	seasons, err := h.seasonService.GetSeasons(r.Context(), seasonEditor(r))
	if errors.Is(err, services.ErrForbidden) {
		webutils.WriteError(w, http.StatusForbidden, "no farm to read seasons for")
		return
	}
	if err != nil {
		slog.Error("getting seasons failed", "error", err)
		webutils.WriteError(w, http.StatusInternalServerError, "internal error")
		return
	}

	webutils.WriteJSON(w, http.StatusOK, toSeasonResponses(seasons))
}

// AssignCropSeason ties a crop to a season: the default rule (admin) or the
// caller's own farm's rule (farmer), creating it if none exists yet or
// repointing it at the given season otherwise. It must be mounted behind
// RequireAuth and RequireAnyRole(models.RoleAdmin, models.RoleFarmer).
func (h *SeasonHandler) AssignCropSeason(w http.ResponseWriter, r *http.Request) {
	cropID, err := uuid.Parse(chi.URLParam(r, "cropID"))
	if err != nil {
		webutils.WriteError(w, http.StatusBadRequest, "invalid crop id")
		return
	}

	var req assignCropSeasonRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		webutils.WriteError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	seasonID, err := uuid.Parse(req.SeasonID)
	if err != nil {
		webutils.WriteError(w, http.StatusBadRequest, "invalid season id")
		return
	}

	rule, err := h.seasonService.AssignCropSeason(r.Context(), seasonEditor(r), cropID, seasonID)
	if errors.Is(err, services.ErrNotFound) {
		webutils.WriteError(w, http.StatusNotFound, "crop or season not found")
		return
	}
	if errors.Is(err, services.ErrForbidden) {
		webutils.WriteError(w, http.StatusForbidden, "season does not belong to the caller's set")
		return
	}
	if err != nil {
		slog.Error("assigning crop season failed", "error", err)
		webutils.WriteError(w, http.StatusInternalServerError, "internal error")
		return
	}

	webutils.WriteJSON(w, http.StatusOK, toCropSeasonRuleResponse(rule))
}

// RemoveCropSeasonRule drops a crop's default rule (admin) or the caller's
// own farm's rule (farmer), reverting it to the default rule if one exists,
// or to unrestricted otherwise. It must be mounted behind RequireAuth and
// RequireAnyRole(models.RoleAdmin, models.RoleFarmer).
func (h *SeasonHandler) RemoveCropSeasonRule(w http.ResponseWriter, r *http.Request) {
	cropID, err := uuid.Parse(chi.URLParam(r, "cropID"))
	if err != nil {
		webutils.WriteError(w, http.StatusBadRequest, "invalid crop id")
		return
	}

	err = h.seasonService.RemoveCropSeasonRule(r.Context(), seasonEditor(r), cropID)
	if errors.Is(err, services.ErrNotFound) {
		webutils.WriteError(w, http.StatusNotFound, "crop has no season rule in the caller's set")
		return
	}
	if errors.Is(err, services.ErrForbidden) {
		webutils.WriteError(w, http.StatusForbidden, "no farm to remove a season rule for")
		return
	}
	if err != nil {
		slog.Error("removing crop season rule failed", "error", err)
		webutils.WriteError(w, http.StatusInternalServerError, "internal error")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// GetCropSeasons returns every crop in the catalog paired with the season it
// is effectively checked against for the caller: the default rule for an
// admin, or the effective rule (their own farm's if it has one, the default
// otherwise) for a farmer. It must be mounted behind RequireAuth and
// RequireAnyRole(models.RoleAdmin, models.RoleFarmer).
func (h *SeasonHandler) GetCropSeasons(w http.ResponseWriter, r *http.Request) {
	cropSeasons, err := h.seasonService.GetCropSeasons(r.Context(), seasonEditor(r))
	if errors.Is(err, services.ErrForbidden) {
		webutils.WriteError(w, http.StatusForbidden, "no farm to read crop seasons for")
		return
	}
	if err != nil {
		slog.Error("getting crop seasons failed", "error", err)
		webutils.WriteError(w, http.StatusInternalServerError, "internal error")
		return
	}

	res := make([]cropSeasonResponse, len(cropSeasons))
	for i, cs := range cropSeasons {
		res[i] = cropSeasonResponse{cropResponse: toCropResponse(cs.Crop)}
		if cs.Season != nil {
			season := toSeasonResponse(*cs.Season)
			res[i].Season = &season
		}
	}

	webutils.WriteJSON(w, http.StatusOK, res)
}
