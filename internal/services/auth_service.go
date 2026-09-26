package services

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/url"
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
	// when it actually means "no such role". Also covers a caller-supplied
	// selector that is invalid on its own terms rather than unowned or
	// missing, e.g. an announcement scoped to both a field and a plot at once.
	ErrInvalidFilter = errors.New("invalid filter")
	// ErrInvalidCareInstruction is returned when a care instruction's week
	// falls outside the range the table accepts. The handler rejects the same
	// input first; this covers the path where the database is the one to
	// notice, so it surfaces as a 400 rather than a 500.
	ErrInvalidCareInstruction = errors.New("invalid care instruction")
	// ErrInvalidRentalRequest is returned when a rental request fails a
	// business rule: a blank message, or a start date outside the window a
	// customer is allowed to request (1 to 60 days out).
	ErrInvalidRentalRequest = errors.New("invalid rental request")
	// ErrRentalAlreadyDecided is returned when approving or declining a
	// rental that is not (or no longer) in the Requested state, including
	// when the id does not exist.
	ErrRentalAlreadyDecided = errors.New("rental already decided")
	// ErrInvalidVerificationToken covers a verify-email token that is
	// malformed, expired, of the wrong type, or whose pending registration
	// has expired or already been consumed. Collapsed into one error, the
	// same way credentials.ErrInvalidToken collapses every JWT failure, so a
	// caller cannot distinguish "expired" from "already used" from "forged".
	ErrInvalidVerificationToken = errors.New("invalid or expired verification token")
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
	// Register validates and stores the registration as pending, then emails
	// a verification link. No account exists until VerifyEmail is called
	// with the token from that email.
	Register(ctx context.Context, input RegisterInput) error
	// VerifyEmail consumes a verification token, creates the account it was
	// issued for, and signs the caller in exactly as Register used to.
	VerifyEmail(ctx context.Context, token string) (models.Account, TokenPair, error)
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
	accountRepo             AccountRepository
	pendingRegistrationRepo PendingRegistrationRepository
	issuer                  *credentials.Issuer
	notificationService     NotificationService
	dispatcher              *Dispatcher
	frontendURL             string
}

func NewAuthService(
	accountRepo AccountRepository,
	pendingRegistrationRepo PendingRegistrationRepository,
	issuer *credentials.Issuer,
	notificationService NotificationService,
	dispatcher *Dispatcher,
	frontendURL string,
) AuthService {
	return &authService{
		accountRepo:             accountRepo,
		pendingRegistrationRepo: pendingRegistrationRepo,
		issuer:                  issuer,
		notificationService:     notificationService,
		dispatcher:              dispatcher,
		frontendURL:             frontendURL,
	}
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

// verifyEmailSubject and verifyEmailTemplate name the outbound message: the
// html/template file in internal/emailtemplates, and its subject line.
const (
	verifyEmailTemplate = "verify_email"
	verifyEmailSubject  = "Bitte bestätige deine E-Mail-Adresse"
)

// verifyEmailData is the payload for the verification email template.
// SendMailFromTemplate (unlike SendMailFromTemplateToMany) executes the
// template against this struct directly, with no enclosing RecipientData --
// so DisplayName lives here rather than behind a .Recipient prefix.
type verifyEmailData struct {
	DisplayName     string
	VerificationURL string
}

// Register validates the input and, instead of creating the account
// immediately, stores it as a PendingRegistration and emails a verification
// link. The account is only created once VerifyEmail consumes that link's
// token — see the type's doc comment for why.
func (s *authService) Register(ctx context.Context, input RegisterInput) error {
	input.FirstName = strings.TrimSpace(input.FirstName)
	input.LastName = strings.TrimSpace(input.LastName)
	input.Email = strings.TrimSpace(input.Email)
	input.FarmName = strings.TrimSpace(input.FarmName)
	input.Address = strings.TrimSpace(input.Address)
	input.Description = strings.TrimSpace(input.Description)

	if input.FirstName == "" || input.LastName == "" || input.Email == "" || input.Password == "" {
		return fmt.Errorf("%w: first name, last name, email and password are required", ErrInvalidRegistration)
	}
	if input.PostalCode <= 0 {
		return fmt.Errorf("%w: a valid postal code is required", ErrInvalidRegistration)
	}
	switch input.Role {
	case models.RoleFarmer:
		if input.FarmName == "" {
			return fmt.Errorf("%w: farm name is required for farmers", ErrInvalidRegistration)
		}
		if input.Address == "" {
			return fmt.Errorf("%w: address is required for farmers", ErrInvalidRegistration)
		}
	case models.RoleCustomer:
		// No role-specific fields to validate.
	default:
		// Admins are seeded directly in the database, never self-registered.
		return fmt.Errorf("%w: role must be farmer or customer", ErrInvalidRegistration)
	}

	// An email already tied to a verified account is taken outright. One
	// still pending is not — UpsertPendingRegistration refreshes it instead,
	// so losing the first verification email does not lock someone out.
	if _, err := s.accountRepo.GetAccountByEmail(ctx, input.Email); err == nil {
		return ErrEmailTaken
	} else if !errors.Is(err, ErrNotFound) {
		return fmt.Errorf("checking existing account: %w", err)
	}

	hash, err := credentials.HashPassword(input.Password)
	if err != nil {
		return fmt.Errorf("hashing password: %w", err)
	}

	pendingID, err := s.pendingRegistrationRepo.UpsertPendingRegistration(ctx, models.PendingRegistration{
		FirstName:    input.FirstName,
		LastName:     input.LastName,
		Email:        input.Email,
		PasswordHash: hash,
		Role:         input.Role,
		FarmName:     input.FarmName,
		Address:      input.Address,
		Description:  input.Description,
		PostalCode:   input.PostalCode,
	}, credentials.EmailVerifyTTL)
	if err != nil {
		return fmt.Errorf("storing pending registration: %w", err)
	}

	token, err := s.issuer.Issue(pendingID, "", credentials.TypeEmailVerify, credentials.EmailVerifyTTL)
	if err != nil {
		return fmt.Errorf("issuing verification token: %w", err)
	}

	verificationURL := s.frontendURL + "/verify-email?token=" + url.QueryEscape(token)
	displayName := input.FirstName + " " + input.LastName

	// Sent off the request goroutine, the same way every other notification
	// in this codebase is: net/smtp has no timeout, so a slow relay must not
	// hold the HTTP response open. A failed send only logs -- the pending
	// registration still exists and a repeat POST /register resends it.
	s.dispatcher.Go(func() {
		data := verifyEmailData{DisplayName: displayName, VerificationURL: verificationURL}
		if err := s.notificationService.SendMailFromTemplate(input.Email, displayName, verifyEmailSubject, verifyEmailTemplate, nil, data); err != nil {
			slog.Error("sending verification email", "error", err, "email", input.Email)
		}
	})

	return nil
}

// VerifyEmail resolves a verification token to its pending registration,
// creates the account, and signs the caller in exactly as Register used to
// before this flow existed.
func (s *authService) VerifyEmail(ctx context.Context, token string) (models.Account, TokenPair, error) {
	claims, err := s.issuer.Parse(token, credentials.TypeEmailVerify)
	if err != nil {
		return models.Account{}, TokenPair{}, ErrInvalidVerificationToken
	}

	pending, err := s.pendingRegistrationRepo.GetPendingRegistrationByID(ctx, claims.UserID)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			return models.Account{}, TokenPair{}, ErrInvalidVerificationToken
		}
		return models.Account{}, TokenPair{}, fmt.Errorf("looking up pending registration: %w", err)
	}

	// The email may have been claimed by someone else (a second, unrelated
	// registration verified first) since this token was issued.
	if _, err := s.accountRepo.GetAccountByEmail(ctx, pending.Email); err == nil {
		return models.Account{}, TokenPair{}, ErrEmailTaken
	} else if !errors.Is(err, ErrNotFound) {
		return models.Account{}, TokenPair{}, fmt.Errorf("checking existing account: %w", err)
	}

	base := models.Account{
		FirstName:    pending.FirstName,
		LastName:     pending.LastName,
		Email:        pending.Email,
		PasswordHash: pending.PasswordHash,
	}

	var account models.Account
	switch pending.Role {
	case models.RoleFarmer:
		account, err = s.accountRepo.CreateFarmer(ctx, base, pending.FarmName, pending.PostalCode, pending.Address, pending.Description)
	case models.RoleCustomer:
		account, err = s.accountRepo.CreateCustomer(ctx, base, pending.PostalCode)
	default:
		// Cannot happen: Register only ever stores Farmer or Customer.
		return models.Account{}, TokenPair{}, fmt.Errorf("pending registration has invalid role %q", pending.Role)
	}
	if err != nil {
		return models.Account{}, TokenPair{}, err
	}

	// Best-effort: the account now exists either way, and a stale pending
	// row is harmless (the email uniqueness check above already keeps it
	// from being used twice) except for cluttering the table until it
	// expires on its own.
	if err := s.pendingRegistrationRepo.DeletePendingRegistration(ctx, pending.ID); err != nil {
		slog.Error("deleting consumed pending registration", "error", err, "id", pending.ID)
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
