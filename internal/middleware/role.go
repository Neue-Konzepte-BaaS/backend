package middleware

import (
	"net/http"

	"github.com/Neue-Konzepte-BaaS/backend/internal/models"
	"github.com/Neue-Konzepte-BaaS/backend/internal/webutils"
)

// RequireRole rejects requests whose authenticated account does not have the
// given role. It must be mounted behind RequireAuth, which puts the claims
// this reads on the request context.
func RequireRole(role models.Role) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			claims := MustClaimsFromContext(r.Context())
			if claims.Role != role {
				webutils.WriteError(w, http.StatusForbidden, "insufficient permissions")
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
