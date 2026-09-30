package middleware

import (
	"log/slog"
	"net/http"

	"github.com/Neue-Konzepte-BaaS/backend/internal/services"
	"github.com/Neue-Konzepte-BaaS/backend/internal/webutils"
)

// RequireActiveSubscription rejects a farmer request unless the caller holds
// an Active or PastDue subscription. It must be mounted behind RequireAuth
// and RequireRole(models.RoleFarmer): it reads claims.UserID assuming both
// already ran. PastDue is treated the same as Active here -- a farmer whose
// renewal payment is mid-retry keeps normal access; only the plot-count cap
// (enforced separately in PlotService) tightens for them.
//
// This is a new kind of gate: unlike RequireRole, which reads only the JWT,
// this makes a database round trip per request. There is no existing
// "account status" concept to extend instead (verification is implicit --
// an unverified farmer has no account row at all, see AuthService.Register),
// so a subscription check has nowhere else to live.
func RequireActiveSubscription(subscriptionService services.SubscriptionService) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			claims := MustClaimsFromContext(r.Context())

			active, err := subscriptionService.HasActiveSubscription(r.Context(), claims.UserID)
			if err != nil {
				slog.Error("checking subscription status failed", "error", err)
				webutils.WriteError(w, http.StatusInternalServerError, "internal error")
				return
			}
			if !active {
				webutils.WriteError(w, http.StatusPaymentRequired, "an active subscription is required")
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
