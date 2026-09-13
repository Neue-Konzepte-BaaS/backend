package services

import (
	"context"
	"errors"
	"fmt"

	"github.com/Neue-Konzepte-BaaS/backend/internal/credentials"
	"github.com/Neue-Konzepte-BaaS/backend/internal/models"
)

var (
	// ErrInvalidCredentials covers both an unknown email and a wrong password.
	ErrInvalidCredentials = errors.New("invalid credentials")
	ErrNotFound           = errors.New("account not found")
)

// dummyHash is verified against when no account matches, so a request for an
// unknown email costs the same time as one for a known email with a bad
// password. Without it, response timing reveals which emails are registered.
var dummyHash, _ = credentials.HashPassword("timing-equalizer")

type TokenPair struct {
	Access  string
	Refresh string
}

type AuthService interface {
	Login(ctx context.Context, email, plainPassword string) (models.Account, TokenPair, error)
	Authenticate(ctx context.Context, accessToken string) (credentials.Claims, error)
}

type authService struct {
	accountRepo AccountRepository
	issuer      *credentials.Issuer
}

func NewAuthService(accountRepo AccountRepository, issuer *credentials.Issuer) AuthService {
	return &authService{accountRepo: accountRepo, issuer: issuer}
}

func (s *authService) Login(ctx context.Context, email, plainPassword string) (models.Account, TokenPair, error) {
	account, err := s.accountRepo.GetAccountByEmail(ctx, email)
	if errors.Is(err, ErrNotFound) {
		_ = credentials.VerifyPassword(plainPassword, dummyHash)
		return models.Account{}, TokenPair{}, ErrInvalidCredentials
	}
	if err != nil {
		return models.Account{}, TokenPair{}, fmt.Errorf("looking up account: %w", err)
	}

	if err := credentials.VerifyPassword(plainPassword, account.PasswordHash); err != nil {
		if errors.Is(err, credentials.ErrPasswordMismatch) {
			return models.Account{}, TokenPair{}, ErrInvalidCredentials
		}
		return models.Account{}, TokenPair{}, fmt.Errorf("verifying password: %w", err)
	}

	pair, err := s.issueTokens(account)
	if err != nil {
		return models.Account{}, TokenPair{}, err
	}

	account.PasswordHash = ""
	return account, pair, nil
}

func (s *authService) Authenticate(ctx context.Context, accessToken string) (credentials.Claims, error) {
	claims, err := s.issuer.Parse(accessToken, credentials.TypeAccess)
	if err != nil {
		return credentials.Claims{}, err
	}

	return claims, nil
}

func (s *authService) issueTokens(account models.Account) (TokenPair, error) {
	access, err := s.issuer.Issue(account.ID, account.Role, credentials.TypeAccess, credentials.AccessTTL)
	if err != nil {
		return TokenPair{}, fmt.Errorf("issuing access token: %w", err)
	}

	refresh, err := s.issuer.Issue(account.ID, account.Role, credentials.TypeRefresh, credentials.RefreshTTL)
	if err != nil {
		return TokenPair{}, fmt.Errorf("issuing refresh token: %w", err)
	}

	return TokenPair{Access: access, Refresh: refresh}, nil
}
