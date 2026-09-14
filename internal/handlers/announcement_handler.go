package handlers

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/Neue-Konzepte-BaaS/backend/internal/middleware"
	"github.com/Neue-Konzepte-BaaS/backend/internal/models"
	"github.com/Neue-Konzepte-BaaS/backend/internal/services"
	"github.com/Neue-Konzepte-BaaS/backend/internal/webutils"
)

type AnnouncementHandler struct {
	announcementService services.AnnouncementService
}

func NewAnnouncementHandler(announcementService services.AnnouncementService) *AnnouncementHandler {
	return &AnnouncementHandler{announcementService: announcementService}
}

// An announcement is mailed to every current renter, so its size is multiplied
// by the audience before it reaches the relay. These bounds are generous for a
// notice a farmer writes by hand and small enough that one post cannot put an
// unbounded payload through the fan-out. They are counted in runes, not bytes:
// a German notice is the normal case, and counting bytes would silently give
// umlauts half the budget.
const (
	maxAnnouncementSubject = 200
	maxAnnouncementBody    = 10_000
)

type createAnnouncementRequest struct {
	Subject string `json:"subject"`
	Body    string `json:"body"`
}

type announcementResponse struct {
	ID        string    `json:"id"`
	Farmer    string    `json:"farmer"`
	FarmName  string    `json:"farm_name"`
	Subject   string    `json:"subject"`
	Body      string    `json:"body"`
	CreatedAt time.Time `json:"created_at"`
}

type createAnnouncementResponse struct {
	announcementResponse
	Recipients int `json:"recipients"`
}

func toAnnouncementResponse(announcement models.AnnouncementWithFarm) announcementResponse {
	return announcementResponse{
		ID:        announcement.ID.String(),
		Farmer:    announcement.Farmer.String(),
		FarmName:  announcement.FarmName,
		Subject:   announcement.Subject,
		Body:      announcement.Body,
		CreatedAt: announcement.CreatedAt,
	}
}

// Create posts an announcement to the farmer's board and mails it to the
// customers currently renting from him. It must be mounted behind RequireAuth
// and RequireRole(models.RoleFarmer).
//
// It answers 201 rather than the 202 the platform broadcast uses: the
// announcement itself is stored before this returns, so something was created.
// Recipients is how many customers were queued, not how many were reached —
// the mail is still going out when the response is written.
func (h *AnnouncementHandler) Create(w http.ResponseWriter, r *http.Request) {
	claims := middleware.MustClaimsFromContext(r.Context())

	var req createAnnouncementRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		webutils.WriteError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	req.Subject = strings.TrimSpace(req.Subject)
	req.Body = strings.TrimSpace(req.Body)
	if req.Subject == "" {
		webutils.WriteError(w, http.StatusBadRequest, "subject is required")
		return
	}
	if req.Body == "" {
		webutils.WriteError(w, http.StatusBadRequest, "body is required")
		return
	}
	if utf8.RuneCountInString(req.Subject) > maxAnnouncementSubject {
		webutils.WriteError(w, http.StatusBadRequest, "subject is too long")
		return
	}
	if utf8.RuneCountInString(req.Body) > maxAnnouncementBody {
		webutils.WriteError(w, http.StatusBadRequest, "body is too long")
		return
	}

	announcement, recipients, err := h.announcementService.CreateAnnouncement(r.Context(), claims.UserID, req.Subject, req.Body)
	if err != nil {
		slog.Error("creating announcement failed", "error", err)
		webutils.WriteError(w, http.StatusInternalServerError, "internal error")
		return
	}

	webutils.WriteJSON(w, http.StatusCreated, createAnnouncementResponse{
		announcementResponse: toAnnouncementResponse(announcement),
		Recipients:           recipients,
	})
}

// List returns the board for whoever is asking. A farmer sees what he has
// posted; a customer sees the notices of every farmer he currently rents from.
// It must be mounted behind RequireAuth and
// RequireAnyRole(models.RoleFarmer, models.RoleCustomer).
func (h *AnnouncementHandler) List(w http.ResponseWriter, r *http.Request) {
	claims := middleware.MustClaimsFromContext(r.Context())

	var (
		announcements []models.AnnouncementWithFarm
		err           error
	)
	switch claims.Role {
	case models.RoleFarmer:
		announcements, err = h.announcementService.GetAnnouncementsForFarmer(r.Context(), claims.UserID)
	case models.RoleCustomer:
		announcements, err = h.announcementService.GetAnnouncementsForCustomer(r.Context(), claims.UserID)
	default:
		// Unreachable behind RequireAnyRole, but a role added later must not
		// silently fall through to an empty board.
		webutils.WriteError(w, http.StatusForbidden, "insufficient permissions")
		return
	}
	if err != nil {
		slog.Error("listing announcements failed", "error", err)
		webutils.WriteError(w, http.StatusInternalServerError, "internal error")
		return
	}

	responses := make([]announcementResponse, len(announcements))
	for i, announcement := range announcements {
		responses[i] = toAnnouncementResponse(announcement)
	}
	webutils.WriteJSON(w, http.StatusOK, responses)
}
