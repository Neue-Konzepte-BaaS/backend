package handlers

import (
	"log/slog"
	"net/http"
	"time"

	"github.com/Neue-Konzepte-BaaS/backend/internal/middleware"
	"github.com/Neue-Konzepte-BaaS/backend/internal/models"
	"github.com/Neue-Konzepte-BaaS/backend/internal/services"
	"github.com/Neue-Konzepte-BaaS/backend/internal/webutils"
)

type InboxHandler struct {
	inboxService services.InboxService
}

func NewInboxHandler(inboxService services.InboxService) *InboxHandler {
	return &InboxHandler{inboxService: inboxService}
}

type inboxItemResponse struct {
	Kind      models.InboxItemKind `json:"kind"`
	ID        string               `json:"id"`
	Subject   string               `json:"subject"`
	Body      string               `json:"body"`
	FarmName  string               `json:"farm_name,omitempty"`
	CreatedAt time.Time            `json:"created_at"`
}

func toInboxItemResponse(item models.InboxItem) inboxItemResponse {
	return inboxItemResponse{
		Kind:      item.Kind,
		ID:        item.ID.String(),
		Subject:   item.Subject,
		Body:      item.Body,
		FarmName:  item.FarmName,
		CreatedAt: item.CreatedAt,
	}
}

// GetInbox returns the caller's merged inbox: every platform-wide broadcast
// plus the announcements of every farmer he currently rents from, newest
// first. It must be mounted behind RequireAuth and RequireRole(models.RoleCustomer).
func (h *InboxHandler) GetInbox(w http.ResponseWriter, r *http.Request) {
	claims := middleware.MustClaimsFromContext(r.Context())

	items, err := h.inboxService.GetInboxForCustomer(r.Context(), claims.UserID)
	if err != nil {
		slog.Error("getting inbox failed", "error", err)
		webutils.WriteError(w, http.StatusInternalServerError, "internal error")
		return
	}

	responses := make([]inboxItemResponse, len(items))
	for i, item := range items {
		responses[i] = toInboxItemResponse(item)
	}
	webutils.WriteJSON(w, http.StatusOK, responses)
}
