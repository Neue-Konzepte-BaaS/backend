package middleware

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Neue-Konzepte-BaaS/backend/internal/credentials"
	"github.com/Neue-Konzepte-BaaS/backend/internal/models"
	"github.com/Neue-Konzepte-BaaS/backend/internal/services"
	"github.com/google/uuid"
)

// stubAuthService records the token it was handed so tests can assert the
// middleware forwards the cookie value verbatim.
type stubAuthService struct {
	claims    credentials.Claims
	err       error
	gotToken  string
	callCount int
}

func (s *stubAuthService) Login(context.Context, string, string) (models.Account, services.TokenPair, error) {
	panic("middleware does not call Login")
}

func (s *stubAuthService) Authenticate(_ context.Context, accessToken string) (credentials.Claims, error) {
	s.gotToken = accessToken
	s.callCount++
	return s.claims, s.err
}

// spyHandler stands in for the wrapped handler, recording whether it ran and
// what context it saw.
type spyHandler struct {
	called bool
	ctx    context.Context
}

func (h *spyHandler) ServeHTTP(_ http.ResponseWriter, r *http.Request) {
	h.called = true
	h.ctx = r.Context()
}

func newRequest(cookie *http.Cookie) *http.Request {
	r := httptest.NewRequest(http.MethodGet, "/protected", nil)
	if cookie != nil {
		r.AddCookie(cookie)
	}
	return r
}

// assertSameClaims compares the identity fields. Claims embeds
// jwt.RegisteredClaims, which holds a slice and so is not comparable with ==.
func assertSameClaims(t *testing.T, got, want credentials.Claims) {
	t.Helper()

	if got.UserID != want.UserID {
		t.Errorf("UserID = %v, want %v", got.UserID, want.UserID)
	}
	if got.Role != want.Role {
		t.Errorf("Role = %q, want %q", got.Role, want.Role)
	}
	if got.Type != want.Type {
		t.Errorf("Type = %q, want %q", got.Type, want.Type)
	}
}

// assertUnauthorized checks the full 401 contract: status, JSON body, and a
// non-committal message that does not reveal why the token was rejected.
func assertUnauthorized(t *testing.T, rec *httptest.ResponseRecorder) {
	t.Helper()

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusUnauthorized)
	}
	if got := rec.Header().Get("Content-Type"); got != "application/json" {
		t.Errorf("Content-Type = %q, want %q", got, "application/json")
	}

	var body map[string]string
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("decoding body %q: %v", rec.Body.String(), err)
	}
	if body["error"] != "not authenticated" {
		t.Errorf("error = %q, want %q", body["error"], "not authenticated")
	}
}

func TestRequireAuthPassesValidTokenThrough(t *testing.T) {
	want := credentials.Claims{
		UserID: uuid.New(),
		Role:   models.RoleFarmer,
		Type:   credentials.TypeAccess,
	}
	auth := &stubAuthService{claims: want}
	next := &spyHandler{}
	rec := httptest.NewRecorder()

	RequireAuth(auth)(next).ServeHTTP(rec, newRequest(&http.Cookie{
		Name:  AccessCookieName,
		Value: "a-valid-token",
	}))

	if !next.called {
		t.Fatal("next handler was not called")
	}
	if auth.gotToken != "a-valid-token" {
		t.Errorf("Authenticate got token %q, want %q", auth.gotToken, "a-valid-token")
	}
	if rec.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusOK)
	}

	got, ok := ClaimsFromContext(next.ctx)
	if !ok {
		t.Fatal("ClaimsFromContext: no claims on the downstream context")
	}
	assertSameClaims(t, got, want)
}

func TestRequireAuthRejectsMissingCookie(t *testing.T) {
	auth := &stubAuthService{claims: credentials.Claims{UserID: uuid.New()}}
	next := &spyHandler{}
	rec := httptest.NewRecorder()

	RequireAuth(auth)(next).ServeHTTP(rec, newRequest(nil))

	if next.called {
		t.Error("next handler ran without an access cookie")
	}
	// The service must not be consulted when there is nothing to verify.
	if auth.callCount != 0 {
		t.Errorf("Authenticate called %d times, want 0", auth.callCount)
	}
	assertUnauthorized(t, rec)
}

// An empty cookie value is rejected before reaching the service, so a client
// cannot probe the verifier with a blank token.
func TestRequireAuthRejectsEmptyCookieValue(t *testing.T) {
	auth := &stubAuthService{claims: credentials.Claims{UserID: uuid.New()}}
	next := &spyHandler{}
	rec := httptest.NewRecorder()

	RequireAuth(auth)(next).ServeHTTP(rec, newRequest(&http.Cookie{
		Name:  AccessCookieName,
		Value: "",
	}))

	if next.called {
		t.Error("next handler ran with an empty access cookie")
	}
	if auth.callCount != 0 {
		t.Errorf("Authenticate called %d times, want 0", auth.callCount)
	}
	assertUnauthorized(t, rec)
}

// A refresh token cookie alone must not authenticate a request.
func TestRequireAuthIgnoresRefreshCookie(t *testing.T) {
	auth := &stubAuthService{claims: credentials.Claims{UserID: uuid.New()}}
	next := &spyHandler{}
	rec := httptest.NewRecorder()

	RequireAuth(auth)(next).ServeHTTP(rec, newRequest(&http.Cookie{
		Name:  RefreshCookieName,
		Value: "a-refresh-token",
	}))

	if next.called {
		t.Error("next handler ran with only a refresh cookie")
	}
	if auth.callCount != 0 {
		t.Errorf("Authenticate called %d times, want 0", auth.callCount)
	}
	assertUnauthorized(t, rec)
}

func TestRequireAuthRejectsInvalidToken(t *testing.T) {
	auth := &stubAuthService{err: credentials.ErrInvalidToken}
	next := &spyHandler{}
	rec := httptest.NewRecorder()

	RequireAuth(auth)(next).ServeHTTP(rec, newRequest(&http.Cookie{
		Name:  AccessCookieName,
		Value: "a-rejected-token",
	}))

	if next.called {
		t.Error("next handler ran with a rejected token")
	}
	assertUnauthorized(t, rec)
}

// Any verification failure, not just an invalid token, must surface as the
// same opaque 401 rather than leaking the underlying error to the client.
func TestRequireAuthHidesServiceErrorDetail(t *testing.T) {
	auth := &stubAuthService{err: errors.New("database connection refused")}
	next := &spyHandler{}
	rec := httptest.NewRecorder()

	RequireAuth(auth)(next).ServeHTTP(rec, newRequest(&http.Cookie{
		Name:  AccessCookieName,
		Value: "a-token",
	}))

	if next.called {
		t.Error("next handler ran after a verification error")
	}
	assertUnauthorized(t, rec)
	if body := rec.Body.String(); strings.Contains(body, "database") {
		t.Errorf("body %q leaks the underlying error", body)
	}
}

func TestClaimsFromContextWithoutClaims(t *testing.T) {
	if _, ok := ClaimsFromContext(context.Background()); ok {
		t.Error("ClaimsFromContext(background) reported claims")
	}
}

// The context key is unexported, so a value stored by another package under a
// lookalike key must not be mistaken for authenticated claims.
func TestClaimsFromContextIgnoresForeignKey(t *testing.T) {
	type contextKey struct{}

	ctx := context.WithValue(context.Background(), contextKey{}, credentials.Claims{
		UserID: uuid.New(),
		Role:   models.RoleAdmin,
	})

	if _, ok := ClaimsFromContext(ctx); ok {
		t.Error("ClaimsFromContext accepted a foreign context key")
	}
}

func TestMustClaimsFromContextReturnsClaims(t *testing.T) {
	want := credentials.Claims{UserID: uuid.New(), Role: models.RoleCustomer}
	ctx := context.WithValue(context.Background(), accountContextKey, want)

	assertSameClaims(t, MustClaimsFromContext(ctx), want)
}

// A handler mounted outside RequireAuth is a routing bug, so it must panic
// (a 500) rather than quietly behave as if the caller were anonymous.
func TestMustClaimsFromContextPanicsWithoutClaims(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Error("MustClaimsFromContext did not panic without claims")
		}
	}()

	MustClaimsFromContext(context.Background())
}
