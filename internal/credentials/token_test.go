package credentials

import (
	"errors"
	"testing"

	"github.com/Neue-Konzepte-BaaS/backend/internal/models"
	"github.com/google/uuid"
)

const testSecret = "test-secret-that-is-long-enough-32"

func TestIssueParseRoundTrip(t *testing.T) {
	issuer := NewIssuer(testSecret)
	id := uuid.New()

	raw, err := issuer.Issue(id, models.RoleFarmer, TypeAccess, AccessTTL)
	if err != nil {
		t.Fatalf("Issue: %v", err)
	}

	claims, err := issuer.Parse(raw, TypeAccess)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if claims.UserID != id {
		t.Errorf("UserID = %v, want %v", claims.UserID, id)
	}
	if claims.Role != models.RoleFarmer {
		t.Errorf("Role = %q, want %q", claims.Role, models.RoleFarmer)
	}
}

// A refresh token must not be accepted where an access token is expected.
func TestParseRejectsWrongType(t *testing.T) {
	issuer := NewIssuer(testSecret)

	refresh, err := issuer.Issue(uuid.New(), models.RoleAdmin, TypeRefresh, RefreshTTL)
	if err != nil {
		t.Fatalf("Issue: %v", err)
	}

	if _, err := issuer.Parse(refresh, TypeAccess); !errors.Is(err, ErrInvalidToken) {
		t.Errorf("Parse(refresh, access) = %v, want ErrInvalidToken", err)
	}
}

func TestParseRejectsForeignSecret(t *testing.T) {
	raw, err := NewIssuer(testSecret).Issue(uuid.New(), models.RoleCustomer, TypeAccess, AccessTTL)
	if err != nil {
		t.Fatalf("Issue: %v", err)
	}

	attacker := NewIssuer("a-completely-different-secret-key!")
	if _, err := attacker.Parse(raw, TypeAccess); !errors.Is(err, ErrInvalidToken) {
		t.Errorf("Parse with wrong secret = %v, want ErrInvalidToken", err)
	}
}

// alg=none must be rejected rather than treated as an unsigned valid token.
func TestParseRejectsAlgNone(t *testing.T) {
	// header {"alg":"none","typ":"JWT"} with an empty signature
	const unsigned = "eyJhbGciOiJub25lIiwidHlwIjoiSldUIn0." +
		"eyJ1c2VyX2lkIjoiMDAwMDAwMDAtMDAwMC0wMDAwLTAwMDAtMDAwMDAwMDAwMDAwIiwidHlwIjoiYWNjZXNzIn0."

	if _, err := NewIssuer(testSecret).Parse(unsigned, TypeAccess); !errors.Is(err, ErrInvalidToken) {
		t.Errorf("Parse(alg=none) = %v, want ErrInvalidToken", err)
	}
}
