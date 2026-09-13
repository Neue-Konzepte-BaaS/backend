// Package middleware holds cross-cutting HTTP concerns. It owns the auth
// cookie names and the request-context account, since the middleware writes
// what handlers read.
package middleware

import (
	"context"
	"net/http"

	"github.com/Neue-Konzepte-BaaS/backend/internal/credentials"
	"github.com/Neue-Konzepte-BaaS/backend/internal/services"
	"github.com/Neue-Konzepte-BaaS/backend/internal/webutils"
)

const (
	AccessCookieName  = "access_token"
	RefreshCookieName = "refresh_token"
)

type contextKey struct{}

var accountContextKey = contextKey{}

// ClaimsFromContext returns the claims placed by RequireAuth, reporting
// whether one was present.
func ClaimsFromContext(ctx context.Context) (credentials.Claims, bool) {
	claims, ok := ctx.Value(accountContextKey).(credentials.Claims)
	return claims, ok
}

// MustClaimsFromContext returns the claims placed by RequireAuth and panics
// if there is none. Use it in handlers mounted behind RequireAuth: a missing
// account means the route is misconfigured, not that the caller is anonymous,
// and that should surface as a 500 rather than a misleading 401.
func MustClaimsFromContext(ctx context.Context) credentials.Claims {
	claims, ok := ClaimsFromContext(ctx)
	if !ok {
		panic("middleware: no account in context; handler is not mounted behind RequireAuth")
	}
	return claims
}

// RequireAuth rejects requests without a valid access token cookie and puts
// the resolved claims on the request context.
func RequireAuth(authService services.AuthService) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			cookie, err := r.Cookie(AccessCookieName)
			if err != nil || cookie.Value == "" {
				webutils.WriteError(w, http.StatusUnauthorized, "not authenticated")
				return
			}

			claims, err := authService.Authenticate(r.Context(), cookie.Value)
			if err != nil {
				webutils.WriteError(w, http.StatusUnauthorized, "not authenticated")
				return
			}

			ctx := context.WithValue(r.Context(), accountContextKey, claims)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
