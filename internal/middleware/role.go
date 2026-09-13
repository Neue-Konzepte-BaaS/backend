package middleware

import (
	"net/http"
	"slices"

	"github.com/Neue-Konzepte-BaaS/backend/internal/models"
	"github.com/Neue-Konzepte-BaaS/backend/internal/webutils"
)

// RequireAnyRole rejects requests whose authenticated account has none of the
// given roles. Passing no roles denies everything. It must be mounted behind
// RequireAuth, which puts the claims this reads on the request context.
func RequireAnyRole(roles ...models.Role) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			claims := MustClaimsFromContext(r.Context())
			if !slices.Contains(roles, claims.Role) {
				webutils.WriteError(w, http.StatusForbidden, "insufficient permissions")
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

// RequireRole is the single-role case of RequireAnyRole.
func RequireRole(role models.Role) func(http.Handler) http.Handler {
	return RequireAnyRole(role)
}
