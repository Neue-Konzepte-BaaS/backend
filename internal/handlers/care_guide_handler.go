package handlers

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
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

type CareGuideHandler struct {
	careGuideService services.CareGuideService
}

func NewCareGuideHandler(careGuideService services.CareGuideService) *CareGuideHandler {
	return &CareGuideHandler{careGuideService: careGuideService}
}

// A care instruction is read on a phone, in a garden, next to the task it
// describes — these bounds keep one week's list scannable there, and match the
// CHECK on care_instruction.week. maxCareWeek is duplicated from the migration
// on purpose: rejecting it here turns a typo into a 400 with a useful message
// instead of a constraint violation.
const (
	maxCareWeek             = 104
	maxCareInstructionTitle = 200
	maxCareInstructionBody  = 5_000
)

type careInstructionRequest struct {
	Week  int32  `json:"week"`
	Title string `json:"title"`
	Body  string `json:"body"`
}

type careInstructionResponse struct {
	ID     string `json:"id"`
	CropID string `json:"cropId"`
	// FarmID is null for a step of the default guide, and the farm's id for
	// a step of that farm's own version.
	FarmID    *string   `json:"farmId"`
	Week      int32     `json:"week"`
	Title     string    `json:"title"`
	Body      string    `json:"body"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

type plotCareGuideResponse struct {
	RentalID     string                    `json:"rentalId"`
	PlotID       string                    `json:"plotId"`
	PlotName     string                    `json:"plotName"`
	FieldName    string                    `json:"fieldName"`
	Crop         cropResponse              `json:"crop"`
	StartAt      string                    `json:"startAt"`
	EndAt        string                    `json:"endAt"`
	CurrentWeek  int32                     `json:"currentWeek"`
	TotalWeeks   int32                     `json:"totalWeeks"`
	Instructions []careInstructionResponse `json:"instructions"`
}

func toCareInstructionResponse(instruction models.CareInstruction) careInstructionResponse {
	var farmID *string
	if instruction.Farm != nil {
		id := instruction.Farm.String()
		farmID = &id
	}
	return careInstructionResponse{
		ID:        instruction.ID.String(),
		CropID:    instruction.Crop.String(),
		FarmID:    farmID,
		Week:      instruction.Week,
		Title:     instruction.Title,
		Body:      instruction.Body,
		CreatedAt: instruction.CreatedAt,
		UpdatedAt: instruction.UpdatedAt,
	}
}

func toCareInstructionResponses(instructions []models.CareInstruction) []careInstructionResponse {
	res := make([]careInstructionResponse, len(instructions))
	for i, instruction := range instructions {
		res[i] = toCareInstructionResponse(instruction)
	}
	return res
}

// careGuideSourceHeader tells a farmer's authoring view which version of the
// guide it is showing — "farm" or "default" — which the instructions alone
// cannot say when the farm's own version is empty. A header rather than a
// wrapping object keeps the body the plain array existing clients read.
const careGuideSourceHeader = "X-Care-Guide-Source"

// careGuideEditor is the caller as the care guide service sees an editor.
func careGuideEditor(r *http.Request) services.CareGuideEditor {
	claims := middleware.MustClaimsFromContext(r.Context())
	return services.CareGuideEditor{AccountID: claims.UserID, Role: claims.Role}
}

// decodeCareInstruction reads and validates the body shared by create and
// update. It writes the error response itself and reports whether the caller
// should carry on.
func decodeCareInstruction(w http.ResponseWriter, r *http.Request) (careInstructionRequest, bool) {
	var req careInstructionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		webutils.WriteError(w, http.StatusBadRequest, "invalid request body")
		return req, false
	}

	req.Title = strings.TrimSpace(req.Title)
	req.Body = strings.TrimSpace(req.Body)

	if req.Week < 1 || req.Week > maxCareWeek {
		webutils.WriteError(w, http.StatusBadRequest, "week must be between 1 and 104")
		return req, false
	}
	if req.Title == "" {
		webutils.WriteError(w, http.StatusBadRequest, "title is required")
		return req, false
	}
	if req.Body == "" {
		webutils.WriteError(w, http.StatusBadRequest, "body is required")
		return req, false
	}
	if utf8.RuneCountInString(req.Title) > maxCareInstructionTitle {
		webutils.WriteError(w, http.StatusBadRequest, "title is too long")
		return req, false
	}
	if utf8.RuneCountInString(req.Body) > maxCareInstructionBody {
		webutils.WriteError(w, http.StatusBadRequest, "body is too long")
		return req, false
	}
	return req, true
}

// CreateCareInstruction adds one task to a crop's guide: the default for an
// admin, the farm's own version for a farmer. It must be mounted behind
// RequireAuth and RequireAnyRole(models.RoleAdmin, models.RoleFarmer).
func (h *CareGuideHandler) CreateCareInstruction(w http.ResponseWriter, r *http.Request) {
	cropID, err := uuid.Parse(chi.URLParam(r, "cropID"))
	if err != nil {
		webutils.WriteError(w, http.StatusBadRequest, "invalid crop id")
		return
	}

	req, ok := decodeCareInstruction(w, r)
	if !ok {
		return
	}

	instruction, err := h.careGuideService.CreateCareInstruction(r.Context(), careGuideEditor(r), cropID, req.Week, req.Title, req.Body)
	if errors.Is(err, services.ErrNotFound) {
		webutils.WriteError(w, http.StatusNotFound, "crop not found")
		return
	}
	if errors.Is(err, services.ErrInvalidCareInstruction) {
		webutils.WriteError(w, http.StatusBadRequest, "week must be between 1 and 104")
		return
	}
	if errors.Is(err, services.ErrForbidden) {
		webutils.WriteError(w, http.StatusForbidden, "no farm to write a care guide for")
		return
	}
	if err != nil {
		slog.Error("creating care instruction failed", "error", err)
		webutils.WriteError(w, http.StatusInternalServerError, "internal error")
		return
	}

	webutils.WriteJSON(w, http.StatusCreated, toCareInstructionResponse(instruction))
}

// UpdateCareInstruction rewrites one task. A farmer editing a step of the
// default guide takes the guide over for their farm, and the response is the
// farm's copy — a different id from the one in the path. It must be mounted
// behind RequireAuth and RequireAnyRole(models.RoleAdmin, models.RoleFarmer).
func (h *CareGuideHandler) UpdateCareInstruction(w http.ResponseWriter, r *http.Request) {
	instructionID, err := uuid.Parse(chi.URLParam(r, "instructionID"))
	if err != nil {
		webutils.WriteError(w, http.StatusBadRequest, "invalid care instruction id")
		return
	}

	req, ok := decodeCareInstruction(w, r)
	if !ok {
		return
	}

	instruction, err := h.careGuideService.UpdateCareInstruction(r.Context(), careGuideEditor(r), instructionID, req.Week, req.Title, req.Body)
	if errors.Is(err, services.ErrNotFound) {
		webutils.WriteError(w, http.StatusNotFound, "care instruction not found")
		return
	}
	if errors.Is(err, services.ErrInvalidCareInstruction) {
		webutils.WriteError(w, http.StatusBadRequest, "week must be between 1 and 104")
		return
	}
	if errors.Is(err, services.ErrForbidden) {
		webutils.WriteError(w, http.StatusForbidden, "no farm to write a care guide for")
		return
	}
	if err != nil {
		slog.Error("updating care instruction failed", "error", err)
		webutils.WriteError(w, http.StatusInternalServerError, "internal error")
		return
	}

	webutils.WriteJSON(w, http.StatusOK, toCareInstructionResponse(instruction))
}

// DeleteCareInstruction removes one task, under the same rules as
// UpdateCareInstruction. It must be mounted behind RequireAuth and
// RequireAnyRole(models.RoleAdmin, models.RoleFarmer).
func (h *CareGuideHandler) DeleteCareInstruction(w http.ResponseWriter, r *http.Request) {
	instructionID, err := uuid.Parse(chi.URLParam(r, "instructionID"))
	if err != nil {
		webutils.WriteError(w, http.StatusBadRequest, "invalid care instruction id")
		return
	}

	err = h.careGuideService.DeleteCareInstruction(r.Context(), careGuideEditor(r), instructionID)
	if errors.Is(err, services.ErrNotFound) {
		webutils.WriteError(w, http.StatusNotFound, "care instruction not found")
		return
	}
	if errors.Is(err, services.ErrForbidden) {
		webutils.WriteError(w, http.StatusForbidden, "no farm to write a care guide for")
		return
	}
	if err != nil {
		slog.Error("deleting care instruction failed", "error", err)
		webutils.WriteError(w, http.StatusInternalServerError, "internal error")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// GetCareInstructions returns one crop's whole guide — the authoring view:
// the default for an admin, and for a farmer the version their tenants read,
// named in the X-Care-Guide-Source header. It must be mounted behind
// RequireAuth and RequireAnyRole(models.RoleAdmin, models.RoleFarmer). A
// customer reads the guide through GetCareGuide instead, where it is scoped to
// what they rent.
func (h *CareGuideHandler) GetCareInstructions(w http.ResponseWriter, r *http.Request) {
	cropID, err := uuid.Parse(chi.URLParam(r, "cropID"))
	if err != nil {
		webutils.WriteError(w, http.StatusBadRequest, "invalid crop id")
		return
	}

	guide, err := h.careGuideService.GetCareInstructionsForCrop(r.Context(), careGuideEditor(r), cropID)
	if errors.Is(err, services.ErrForbidden) {
		webutils.WriteError(w, http.StatusForbidden, "no farm to read a care guide for")
		return
	}
	if err != nil {
		slog.Error("getting care instructions failed", "error", err)
		webutils.WriteError(w, http.StatusInternalServerError, "internal error")
		return
	}

	source := "default"
	if guide.FarmGuide {
		source = "farm"
	}
	w.Header().Set(careGuideSourceHeader, source)
	webutils.WriteJSON(w, http.StatusOK, toCareInstructionResponses(guide.Instructions))
}

// ResetFarmCareGuide drops the calling farmer's own version of a crop's guide,
// so their tenants read the default again. It must be mounted behind
// RequireAuth and RequireRole(models.RoleFarmer).
func (h *CareGuideHandler) ResetFarmCareGuide(w http.ResponseWriter, r *http.Request) {
	claims := middleware.MustClaimsFromContext(r.Context())

	cropID, err := uuid.Parse(chi.URLParam(r, "cropID"))
	if err != nil {
		webutils.WriteError(w, http.StatusBadRequest, "invalid crop id")
		return
	}

	err = h.careGuideService.ResetFarmCareGuide(r.Context(), claims.UserID, cropID)
	if errors.Is(err, services.ErrNotFound) {
		webutils.WriteError(w, http.StatusNotFound, "farm has no care guide of its own for this crop")
		return
	}
	if errors.Is(err, services.ErrForbidden) {
		webutils.WriteError(w, http.StatusForbidden, "no farm to reset a care guide for")
		return
	}
	if err != nil {
		slog.Error("resetting farm care guide failed", "error", err)
		webutils.WriteError(w, http.StatusInternalServerError, "internal error")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// GetCareGuide returns the care guide for each plot the authenticated
// customer is currently renting. It must be mounted behind RequireAuth and
// RequireRole(models.RoleCustomer).
func (h *CareGuideHandler) GetCareGuide(w http.ResponseWriter, r *http.Request) {
	claims := middleware.MustClaimsFromContext(r.Context())

	guides, err := h.careGuideService.GetCareGuideForCustomer(r.Context(), claims.UserID)
	if err != nil {
		slog.Error("getting care guide failed", "error", err)
		webutils.WriteError(w, http.StatusInternalServerError, "internal error")
		return
	}

	res := make([]plotCareGuideResponse, len(guides))
	for i, guide := range guides {
		res[i] = plotCareGuideResponse{
			RentalID:     guide.RentalID.String(),
			PlotID:       guide.PlotID.String(),
			PlotName:     guide.PlotName,
			FieldName:    guide.FieldName,
			Crop:         toCropResponse(guide.Crop),
			StartAt:      guide.StartAt.Format(time.RFC3339),
			EndAt:        guide.EndAt.Format(time.RFC3339),
			CurrentWeek:  guide.CurrentWeek,
			TotalWeeks:   guide.TotalWeeks,
			Instructions: toCareInstructionResponses(guide.Instructions),
		}
	}

	webutils.WriteJSON(w, http.StatusOK, res)
}
