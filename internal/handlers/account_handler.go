package handlers

import (
	"errors"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/Neue-Konzepte-BaaS/backend/internal/middleware"
	"github.com/Neue-Konzepte-BaaS/backend/internal/models"
	"github.com/Neue-Konzepte-BaaS/backend/internal/services"
	"github.com/Neue-Konzepte-BaaS/backend/internal/webutils"
)

type AccountHandler struct {
	accountService services.AccountService
}

func NewAccountHandler(accountService services.AccountService) *AccountHandler {
	return &AccountHandler{accountService: accountService}
}

type accountListingResponse struct {
	ID        string `json:"id"`
	FirstName string `json:"firstName"`
	LastName  string `json:"lastName"`
	Email     string `json:"email"`
	// Role is null for an account with no subtype row. Null rather than "" so
	// the orphan is visibly an orphan instead of looking like a role the
	// client failed to recognise.
	Role      *string `json:"role"`
	CreatedAt string  `json:"createdAt"`
}

type accountPageResponse struct {
	Items  []accountListingResponse `json:"items"`
	Total  int64                    `json:"total"`
	Limit  int32                    `json:"limit"`
	Offset int32                    `json:"offset"`
}

// ListAccounts returns one page of every account on the platform. It must be
// mounted behind RequireAuth and RequireRole(models.RoleAdmin).
func (h *AccountHandler) ListAccounts(w http.ResponseWriter, r *http.Request) {
	claims := middleware.MustClaimsFromContext(r.Context())
	query := r.URL.Query()

	limit, err := parseLimit(query.Get("limit"), defaultPageLimit, maxPageLimit)
	if err != nil {
		webutils.WriteError(w, http.StatusBadRequest, "limit must be a number between 1 and 100")
		return
	}
	offset, err := parseOffset(query.Get("offset"))
	if err != nil {
		webutils.WriteError(w, http.StatusBadRequest, "offset must be a number of 0 or more")
		return
	}

	page, err := h.accountService.ListAccounts(r.Context(), claims.Role, models.AccountListFilter{
		Role:   models.Role(strings.TrimSpace(query.Get("role"))),
		Query:  query.Get("q"),
		Limit:  limit,
		Offset: offset,
	})
	if errors.Is(err, services.ErrInvalidFilter) {
		webutils.WriteError(w, http.StatusBadRequest, "role must be one of admin, farmer, customer")
		return
	}
	if errors.Is(err, services.ErrForbidden) {
		webutils.WriteError(w, http.StatusForbidden, "insufficient permissions")
		return
	}
	if err != nil {
		slog.Error("listing accounts failed", "error", err)
		webutils.WriteError(w, http.StatusInternalServerError, "internal error")
		return
	}

	webutils.WriteJSON(w, http.StatusOK, toAccountPageResponse(page))
}

func toAccountPageResponse(page models.Page[models.AccountListing]) accountPageResponse {
	items := make([]accountListingResponse, len(page.Items))
	for i, account := range page.Items {
		items[i] = accountListingResponse{
			ID:        account.ID.String(),
			FirstName: account.FirstName,
			LastName:  account.LastName,
			Email:     account.Email,
			Role:      optionalRole(account.Role),
			CreatedAt: account.CreatedAt.Format(time.RFC3339),
		}
	}
	return accountPageResponse{
		Items:  items,
		Total:  page.Total,
		Limit:  page.Limit,
		Offset: page.Offset,
	}
}

func optionalRole(role models.Role) *string {
	if role == "" {
		return nil
	}
	name := string(role)
	return &name
}
