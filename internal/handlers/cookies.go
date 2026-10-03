package handlers

import (
	"net/http"

	"github.com/Neue-Konzepte-BaaS/backend/internal/config"
	"github.com/Neue-Konzepte-BaaS/backend/internal/middleware"
)

// clearAuthCookies overwrites both auth cookies with expired ones, ending
// the session client-side. The tokens are HttpOnly, so the browser can't
// clear them itself. Shared by AuthHandler.Logout and
// AccountHandler.DeleteMyAccount, since both end a session the same way.
func clearAuthCookies(w http.ResponseWriter, cfg config.Config) {
	clearCookie(w, cfg, middleware.AccessCookieName, "/")
	clearCookie(w, cfg, middleware.RefreshCookieName, "/api/auth/refresh")
}

// clearCookie overwrites a cookie with an expired one. The attributes
// (Path, Secure, SameSite) must match the original for the browser to
// replace it.
func clearCookie(w http.ResponseWriter, cfg config.Config, name, path string) {
	sameSite := http.SameSiteLaxMode
	if cfg.SameSiteStrict {
		sameSite = http.SameSiteStrictMode
	}

	http.SetCookie(w, &http.Cookie{
		Name:     name,
		Value:    "",
		Path:     path,
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   cfg.CookieSecure,
		SameSite: sameSite,
	})
}
