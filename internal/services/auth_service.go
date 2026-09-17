package services

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/Neue-Konzepte-BaaS/backend/internal/credentials"
	"github.com/Neue-Konzepte-BaaS/backend/internal/models"
	"github.com/google/uuid"
)

var (
	// ErrInvalidCredentials covers both an unknown email and a wrong password.
	ErrInvalidCredentials = errors.New("invalid credentials")
	ErrNotFound           = errors.New("account not found")
	// ErrEmailTaken is returned when registering with an email that already exists.
	ErrEmailTaken = errors.New("email already registered")
	// ErrInvalidGeometry covers rejected field/plot coordinates: not a
	// rectangle, or (for plots) not contained within the parent field.
	ErrInvalidGeometry = errors.New("invalid geometry")
	// ErrForbidden covers an authenticated account acting on a resource it
	// does not own, e.g. a farmer creating a plot on another farmer's field.
	ErrForbidden = errors.New("forbidden")
	// ErrPlotUnavailable is returned when a plot is already rented for the
	// requested period.
	ErrPlotUnavailable = errors.New("plot unavailable")
	// ErrCropNotOffered is returned when renting a plot with a crop it does
	// not offer.
	ErrCropNotOffered = errors.New("crop not offered by this plot")
	// ErrCropNameTaken is returned when creating a crop whose name is already
	// in the catalog.
	ErrCropNameTaken = errors.New("crop name already exists")
	// ErrConflict is returned when an operation would violate a referential
	// constraint, e.g. deleting a crop still referenced by a rental.
	ErrConflict = errors.New("conflict")
	// ErrInvalidFilter is returned when a listing filter cannot be satisfied
	// as written, e.g. a role that is not one of the three. Returning it beats
	// silently answering with an empty page, which reads as "no such accounts"
	// when it actually means "no such role".
	ErrInvalidFilter = errors.New("invalid filter")
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
	Register(ctx context.Context, input RegisterInput) (models.Account, TokenPair, error)
	Authenticate(ctx context.Context, accessToken string) (credentials.Claims, error)
	// Me resolves the full account behind an already-authenticated request
	// (see middleware.RequireAuth). The JWT claims alone only carry the user
	// id and role, not name/postal code, so /auth/me needs this extra lookup
	// to answer with more than that.
	Me(ctx context.Context, id uuid.UUID) (models.Account, error)
}

// ErrInvalidRegistration reports a registration that fails a business rule
// (missing field, unsupported role). Handlers map it to 400.
var ErrInvalidRegistration = errors.New("invalid registration")

// RegisterInput is the validated shape the service needs to create an account.
// Role must be farmer or customer; admins are created by DB seed, not here.
// FarmName, Address and Description apply only to farmers; PostalCode applies
// to both.
type RegisterInput struct {
	FirstName   string
	LastName    string
	Email       string
	Password    string
	Role        models.Role
	FarmName    string
	Address     string
	Description string
	PostalCode  int32
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

// Register creates a farmer or customer account, then issues the same token
// pair as Login so the caller is signed in immediately after registering.
func (s *authService) Register(ctx context.Context, input RegisterInput) (models.Account, TokenPair, error) {
	input.FirstName = strings.TrimSpace(input.FirstName)
	input.LastName = strings.TrimSpace(input.LastName)
	input.Email = strings.TrimSpace(input.Email)
	input.FarmName = strings.TrimSpace(input.FarmName)
	input.Address = strings.TrimSpace(input.Address)
	input.Description = strings.TrimSpace(input.Description)

	if input.FirstName == "" || input.LastName == "" || input.Email == "" || input.Password == "" {
		return models.Account{}, TokenPair{}, fmt.Errorf("%w: first name, last name, email and password are required", ErrInvalidRegistration)
	}
	if input.PostalCode <= 0 {
		return models.Account{}, TokenPair{}, fmt.Errorf("%w: a valid postal code is required", ErrInvalidRegistration)
	}

	hash, err := credentials.HashPassword(input.Password)
	if err != nil {
		return models.Account{}, TokenPair{}, fmt.Errorf("hashing password: %w", err)
	}

	base := models.Account{
		FirstName:    input.FirstName,
		LastName:     input.LastName,
		Email:        input.Email,
		PasswordHash: hash,
	}

	var account models.Account
	switch input.Role {
	case models.RoleFarmer:
		if input.FarmName == "" {
			return models.Account{}, TokenPair{}, fmt.Errorf("%w: farm name is required for farmers", ErrInvalidRegistration)
		}
		if input.Address == "" {
			return models.Account{}, TokenPair{}, fmt.Errorf("%w: address is required for farmers", ErrInvalidRegistration)
		}
		account, err = s.accountRepo.CreateFarmer(ctx, base, input.FarmName, input.PostalCode, input.Address, input.Description)
	case models.RoleCustomer:
		account, err = s.accountRepo.CreateCustomer(ctx, base, input.PostalCode)
	default:
		// Admins are seeded directly in the database, never self-registered.
		return models.Account{}, TokenPair{}, fmt.Errorf("%w: role must be farmer or customer", ErrInvalidRegistration)
	}
	if err != nil {
		return models.Account{}, TokenPair{}, err
	}

	pair, err := s.issueTokens(account)
	if err != nil {
		return models.Account{}, TokenPair{}, err
	}

	account.PasswordHash = ""
	return account, pair, nil
}

func (s *authService) Me(ctx context.Context, id uuid.UUID) (models.Account, error) {
	account, err := s.accountRepo.GetAccountByID(ctx, id)
	if err != nil {
		// Wrapped, not returned bare: errors.Is still finds ErrNotFound through
		// %w, and everything else keeps the "where did this fail" context.
		return models.Account{}, fmt.Errorf("resolving account: %w", err)
	}

	account.PasswordHash = ""
	return account, nil
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
