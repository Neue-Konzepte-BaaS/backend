package credentials

import (
	"errors"
	"fmt"
	"time"

	"github.com/Neue-Konzepte-BaaS/backend/internal/models"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

// ErrInvalidToken covers every rejection reason (bad signature, expired,
// wrong type) so callers cannot distinguish them and leak detail to clients.
var ErrInvalidToken = errors.New("invalid token")

const (
	AccessTTL  = 15 * time.Minute
	RefreshTTL = 7 * 24 * time.Hour

	TypeAccess  = "access"
	TypeRefresh = "refresh"
)

// Claims carries the user identity. Role is included so authorization checks
// do not need a database round trip.
type Claims struct {
	UserID uuid.UUID   `json:"user_id"`
	Role   models.Role `json:"role"`
	Type   string      `json:"typ"`
	jwt.RegisteredClaims
}

type Issuer struct {
	secret []byte
}

func NewIssuer(secret string) *Issuer {
	return &Issuer{secret: []byte(secret)}
}

func (i *Issuer) Issue(userID uuid.UUID, role models.Role, tokenType string, ttl time.Duration) (string, error) {
	now := time.Now()
	claims := Claims{
		UserID: userID,
		Role:   role,
		Type:   tokenType,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   userID.String(),
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(ttl)),
		},
	}

	signed, err := jwt.NewWithClaims(jwt.SigningMethodRS256, claims).SignedString(i.secret)
	if err != nil {
		return "", fmt.Errorf("signing %s token: %w", tokenType, err)
	}
	return signed, nil
}

// Parse validates the token's signature and expiry and checks that it is of
// the expected type, so a refresh token cannot be used to access a resource.
func (i *Issuer) Parse(raw, expectedType string) (Claims, error) {
	var claims Claims
	_, err := jwt.ParseWithClaims(raw, &claims, func(t *jwt.Token) (any, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method %v", t.Header["alg"])
		}
		return i.secret, nil
	}, jwt.WithValidMethods([]string{jwt.SigningMethodRS256.Alg()}))
	if err != nil {
		return Claims{}, ErrInvalidToken
	}

	if claims.Type != expectedType {
		return Claims{}, ErrInvalidToken
	}

	return claims, nil
}
