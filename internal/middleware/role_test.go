package middleware

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Neue-Konzepte-BaaS/backend/internal/credentials"
	"github.com/Neue-Konzepte-BaaS/backend/internal/models"
	"github.com/google/uuid"
)

// requestWithRole builds a request carrying claims for the given role,
// injected directly into the context the way RequireAuth would have.
func requestWithRole(role models.Role) *http.Request {
	ctx := context.WithValue(context.Background(), accountContextKey, credentials.Claims{
		UserID: uuid.New(),
		Role:   role,
	})
	return httptest.NewRequest(http.MethodGet, "/protected", nil).WithContext(ctx)
}

// assertForbidden checks the full 403 contract: status and JSON body.
func assertForbidden(t *testing.T, rec *httptest.ResponseRecorder) {
	t.Helper()

	if rec.Code != http.StatusForbidden {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusForbidden)
	}

	var body map[string]string
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("decoding body %q: %v", rec.Body.String(), err)
	}
	if body["error"] != "insufficient permissions" {
		t.Errorf("error = %q, want %q", body["error"], "insufficient permissions")
	}
}

func TestRequireAnyRole_AdmitsEachListedRole(t *testing.T) {
	for _, role := range []models.Role{models.RoleFarmer, models.RoleAdmin} {
		t.Run(string(role), func(t *testing.T) {
			next := &spyHandler{}
			rec := httptest.NewRecorder()

			RequireAnyRole(models.RoleFarmer, models.RoleAdmin)(next).ServeHTTP(rec, requestWithRole(role))

			if !next.called {
				t.Fatal("next handler was not called")
			}
			if rec.Code != http.StatusOK {
				t.Errorf("status = %d, want %d", rec.Code, http.StatusOK)
			}
		})
	}
}

func TestRequireAnyRole_RejectsUnlistedRole(t *testing.T) {
	next := &spyHandler{}
	rec := httptest.NewRecorder()

	RequireAnyRole(models.RoleFarmer, models.RoleAdmin)(next).ServeHTTP(rec, requestWithRole(models.RoleCustomer))

	if next.called {
		t.Error("next handler ran for a role not in the allow-list")
	}
	assertForbidden(t, rec)
}

// Passing no roles denies everything -- fail closed, not open.
func TestRequireAnyRole_NoRolesDeniesEverything(t *testing.T) {
	next := &spyHandler{}
	rec := httptest.NewRecorder()

	RequireAnyRole()(next).ServeHTTP(rec, requestWithRole(models.RoleAdmin))

	if next.called {
		t.Error("next handler ran with an empty role list")
	}
	assertForbidden(t, rec)
}

// RequireRole must still behave as the single-role case, proving the
// RequireAnyRole rewrite did not widen any existing route's access.
func TestRequireRole_StillRejectsOtherRoles(t *testing.T) {
	next := &spyHandler{}
	rec := httptest.NewRecorder()

	RequireRole(models.RoleFarmer)(next).ServeHTTP(rec, requestWithRole(models.RoleAdmin))

	if next.called {
		t.Error("next handler ran for a role other than the single required one")
	}
	assertForbidden(t, rec)
}

func TestRequireRole_AdmitsMatchingRole(t *testing.T) {
	next := &spyHandler{}
	rec := httptest.NewRecorder()

	RequireRole(models.RoleFarmer)(next).ServeHTTP(rec, requestWithRole(models.RoleFarmer))

	if !next.called {
		t.Error("next handler did not run for the matching role")
	}
}
