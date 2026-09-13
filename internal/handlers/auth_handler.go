package handlers

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/Neue-Konzepte-BaaS/backend/internal/config"
	"github.com/Neue-Konzepte-BaaS/backend/internal/credentials"
	"github.com/Neue-Konzepte-BaaS/backend/internal/middleware"
	"github.com/Neue-Konzepte-BaaS/backend/internal/models"
	"github.com/Neue-Konzepte-BaaS/backend/internal/services"
	"github.com/Neue-Konzepte-BaaS/backend/internal/webutils"
)

type AuthHandler struct {
	authService services.AuthService
	cfg         config.Config
}

func NewAuthHandler(authService services.AuthService, cfg config.Config) *AuthHandler {
	return &AuthHandler{authService: authService, cfg: cfg}
}

type loginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type registerRequest struct {
	FirstName  string `json:"first_name"`
	LastName   string `json:"last_name"`
	Email      string `json:"email"`
	Password   string `json:"password"`
	Role       string `json:"role"`
	FarmName   string `json:"farm_name"`
	PostalCode int32  `json:"postal_code"`
}

type meResponse struct {
	ID   string `json:"id"`
	Role string `json:"role"`
}

func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	var req loginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		webutils.WriteError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	req.Email = strings.TrimSpace(req.Email)
	if req.Email == "" || req.Password == "" {
		webutils.WriteError(w, http.StatusBadRequest, "email and password are required")
		return
	}

	account, pair, err := h.authService.Login(r.Context(), req.Email, req.Password)
	if errors.Is(err, services.ErrInvalidCredentials) {
		webutils.WriteError(w, http.StatusUnauthorized, "invalid credentials")
		return
	}
	if err != nil {
		slog.Error("login failed", "error", err)
		webutils.WriteError(w, http.StatusInternalServerError, "internal error")
		return
	}

	h.setCookie(w, middleware.AccessCookieName, pair.Access, "/", credentials.AccessTTL)
	h.setCookie(w, middleware.RefreshCookieName, pair.Refresh, "/api/auth/refresh", credentials.RefreshTTL)

	webutils.WriteJSON(w, http.StatusOK, meResponse{
		ID:   account.ID.String(),
		Role: string(account.Role),
	})
}

// Register creates a farmer or customer account and, on success, signs the new
// user in by setting the same auth cookies as Login.
func (h *AuthHandler) Register(w http.ResponseWriter, r *http.Request) {
	var req registerRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		webutils.WriteError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	account, pair, err := h.authService.Register(r.Context(), services.RegisterInput{
		FirstName:  req.FirstName,
		LastName:   req.LastName,
		Email:      req.Email,
		Password:   req.Password,
		Role:       models.Role(strings.TrimSpace(req.Role)),
		FarmName:   req.FarmName,
		PostalCode: req.PostalCode,
	})
	if errors.Is(err, services.ErrInvalidRegistration) {
		// Strip the sentinel prefix so the client sees only the human-readable detail.
		msg := strings.TrimPrefix(err.Error(), services.ErrInvalidRegistration.Error()+": ")
		webutils.WriteError(w, http.StatusBadRequest, msg)
		return
	}
	if errors.Is(err, services.ErrEmailTaken) {
		webutils.WriteError(w, http.StatusConflict, "email already registered")
		return
	}
	if err != nil {
		slog.Error("registration failed", "error", err)
		webutils.WriteError(w, http.StatusInternalServerError, "internal error")
		return
	}

	h.setCookie(w, middleware.AccessCookieName, pair.Access, "/", credentials.AccessTTL)
	h.setCookie(w, middleware.RefreshCookieName, pair.Refresh, "/api/auth/refresh", credentials.RefreshTTL)

	webutils.WriteJSON(w, http.StatusCreated, meResponse{
		ID:   account.ID.String(),
		Role: string(account.Role),
	})
}

// Me reports the authenticated account. It must be mounted behind RequireAuth.
func (h *AuthHandler) Me(w http.ResponseWriter, r *http.Request) {	accountClaims := middleware.MustClaimsFromContext(r.Context())

	webutils.WriteJSON(w, http.StatusOK, meResponse{
		ID:   accountClaims.UserID.String(),
		Role: string(accountClaims.Role),
	})
}

func (h *AuthHandler) setCookie(w http.ResponseWriter, name, value, path string, ttl time.Duration) {
	sameSite := http.SameSiteLaxMode
	if h.cfg.SameSiteStrict {
		sameSite = http.SameSiteStrictMode
	}

	http.SetCookie(w, &http.Cookie{
		Name:     name,
		Value:    value,
		Path:     path,
		MaxAge:   int(ttl.Seconds()),
		HttpOnly: true,
		Secure:   h.cfg.CookieSecure,
		SameSite: sameSite,
	})
}
