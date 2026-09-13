package handlers

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"strings"

	"github.com/Neue-Konzepte-BaaS/backend/internal/services"
	"github.com/Neue-Konzepte-BaaS/backend/internal/webutils"
)

type NotificationHandler struct {
	notificationService services.NotificationService
}

func NewNotificationHandler(notificationService services.NotificationService) *NotificationHandler {
	return &NotificationHandler{notificationService: notificationService}
}

type broadcastRequest struct {
	Subject string `json:"subject"`
	Body    string `json:"body"`
}

type broadcastResponse struct {
	Recipients int `json:"recipients"`
}

// Broadcast sends a message to every farmer and customer on the platform. It
// must be mounted behind RequireAuth and RequireRole(models.RoleAdmin).
//
// It answers 202 rather than 201: nothing is persisted, and the mail is still
// being delivered when the response is written, so Recipients is how many
// people were queued — not how many were reached.
func (h *NotificationHandler) Broadcast(w http.ResponseWriter, r *http.Request) {
	var req broadcastRequest
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

	recipients, err := h.notificationService.NotifyAllUsers(r.Context(), req.Subject, req.Body)
	if err != nil {
		slog.Error("broadcasting notification failed", "error", err)
		webutils.WriteError(w, http.StatusInternalServerError, "internal error")
		return
	}

	webutils.WriteJSON(w, http.StatusAccepted, broadcastResponse{Recipients: recipients})
}
