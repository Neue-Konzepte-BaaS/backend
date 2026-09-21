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
	ID        string    `json:"id"`
	CropID    string    `json:"cropId"`
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
	return careInstructionResponse{
		ID:        instruction.ID.String(),
		CropID:    instruction.Crop.String(),
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

// CreateCareInstruction adds one task to a crop's guide. It must be mounted
// behind RequireAuth and RequireRole(models.RoleAdmin).
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

	instruction, err := h.careGuideService.CreateCareInstruction(r.Context(), cropID, req.Week, req.Title, req.Body)
	if errors.Is(err, services.ErrNotFound) {
		webutils.WriteError(w, http.StatusNotFound, "crop not found")
		return
	}
	if errors.Is(err, services.ErrInvalidCareInstruction) {
		webutils.WriteError(w, http.StatusBadRequest, "week must be between 1 and 104")
		return
	}
	if err != nil {
		slog.Error("creating care instruction failed", "error", err)
		webutils.WriteError(w, http.StatusInternalServerError, "internal error")
		return
	}

	webutils.WriteJSON(w, http.StatusCreated, toCareInstructionResponse(instruction))
}

// UpdateCareInstruction rewrites one task. It must be mounted behind
// RequireAuth and RequireRole(models.RoleAdmin).
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

	instruction, err := h.careGuideService.UpdateCareInstruction(r.Context(), instructionID, req.Week, req.Title, req.Body)
	if errors.Is(err, services.ErrNotFound) {
		webutils.WriteError(w, http.StatusNotFound, "care instruction not found")
		return
	}
	if errors.Is(err, services.ErrInvalidCareInstruction) {
		webutils.WriteError(w, http.StatusBadRequest, "week must be between 1 and 104")
		return
	}
	if err != nil {
		slog.Error("updating care instruction failed", "error", err)
		webutils.WriteError(w, http.StatusInternalServerError, "internal error")
		return
	}

	webutils.WriteJSON(w, http.StatusOK, toCareInstructionResponse(instruction))
}

// DeleteCareInstruction removes one task. It must be mounted behind
// RequireAuth and RequireRole(models.RoleAdmin).
func (h *CareGuideHandler) DeleteCareInstruction(w http.ResponseWriter, r *http.Request) {
	instructionID, err := uuid.Parse(chi.URLParam(r, "instructionID"))
	if err != nil {
		webutils.WriteError(w, http.StatusBadRequest, "invalid care instruction id")
		return
	}

	err = h.careGuideService.DeleteCareInstruction(r.Context(), instructionID)
	if errors.Is(err, services.ErrNotFound) {
		webutils.WriteError(w, http.StatusNotFound, "care instruction not found")
		return
	}
	if err != nil {
		slog.Error("deleting care instruction failed", "error", err)
		webutils.WriteError(w, http.StatusInternalServerError, "internal error")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// GetCareInstructions returns one crop's whole guide — the authoring and
// preview view. It must be mounted behind RequireAuth and
// RequireAnyRole(models.RoleAdmin, models.RoleFarmer): an admin writes the
// guide, a farmer reads what his renters will be told. A customer reads the
// guide through GetCareGuide instead, where it is scoped to what they rent.
func (h *CareGuideHandler) GetCareInstructions(w http.ResponseWriter, r *http.Request) {
	cropID, err := uuid.Parse(chi.URLParam(r, "cropID"))
	if err != nil {
		webutils.WriteError(w, http.StatusBadRequest, "invalid crop id")
		return
	}

	instructions, err := h.careGuideService.GetCareInstructionsForCrop(r.Context(), cropID)
	if err != nil {
		slog.Error("getting care instructions failed", "error", err)
		webutils.WriteError(w, http.StatusInternalServerError, "internal error")
		return
	}

	webutils.WriteJSON(w, http.StatusOK, toCareInstructionResponses(instructions))
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
