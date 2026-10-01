package services

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"

	"github.com/Neue-Konzepte-BaaS/backend/internal/credentials"
	"github.com/Neue-Konzepte-BaaS/backend/internal/models"
	"github.com/google/uuid"
)

var (
	// ErrAccountHasOpenRentals is returned when deleting an account that
	// still has a requested or approved rental that has not run out -- the
	// customer's own, or one on the farmer's plots. Someone has paid for it.
	ErrAccountHasOpenRentals = errors.New("account has open rentals")
	// ErrAccountHasPendingPayment is returned when deleting an account while
	// one of its Stripe checkouts has not settled yet.
	ErrAccountHasPendingPayment = errors.New("account has a pending payment")
)

type AccountService interface {
	// ListAccounts returns one page of the platform's accounts. Only an admin
	// may call it; every other role gets ErrForbidden. A Role filter that is
	// not one of the three known roles is ErrInvalidFilter.
	ListAccounts(ctx context.Context, role models.Role, filter models.AccountListFilter) (models.Page[models.AccountListing], error)
	// DeleteAccount deletes the caller's own account after re-checking their
	// password (GDPR Art. 17). A wrong password is ErrInvalidCredentials; an
	// admin gets ErrForbidden, since admins are managed in the database. An
	// account still bound to a rental or an unsettled payment is
	// ErrAccountHasOpenRentals or ErrAccountHasPendingPayment. A farmer's
	// running subscription is canceled in Stripe on the way out.
	DeleteAccount(ctx context.Context, id uuid.UUID, role models.Role, password string) error
}

type accountService struct {
	accountRepo    AccountRepository
	farmerSubRepo  FarmerSubscriptionRepository
	paymentGateway PaymentGateway
}

func NewAccountService(accountRepo AccountRepository, farmerSubRepo FarmerSubscriptionRepository, paymentGateway PaymentGateway) AccountService {
	return &accountService{accountRepo: accountRepo, farmerSubRepo: farmerSubRepo, paymentGateway: paymentGateway}
}

func (s *accountService) ListAccounts(ctx context.Context, role models.Role, filter models.AccountListFilter) (models.Page[models.AccountListing], error) {
	// The route is gated by RequireRole(admin), so this is defence in depth --
	// the same belt-and-braces as statisticsService.GetStatistics. The scope is
	// the caller's role and nothing else: there is no parameter that widens it.
	if role != models.RoleAdmin {
		return models.Page[models.AccountListing]{}, ErrForbidden
	}

	filter.Query = strings.TrimSpace(filter.Query)
	if filter.Role != "" && !slices.Contains(knownRoles, filter.Role) {
		// An unknown role would match nothing and come back as an empty page,
		// which reads as "there are no such accounts" rather than "there is no
		// such role". Say which it is.
		return models.Page[models.AccountListing]{}, fmt.Errorf("listing accounts: role %q: %w", filter.Role, ErrInvalidFilter)
	}
	filter.Limit, filter.Offset = clampPagination(filter.Limit, filter.Offset)

	page, err := s.accountRepo.ListAccounts(ctx, filter)
	if err != nil {
		return models.Page[models.AccountListing]{}, fmt.Errorf("listing accounts: %w", err)
	}
	return page, nil
}

func (s *accountService) DeleteAccount(ctx context.Context, id uuid.UUID, role models.Role, password string) error {
	if role != models.RoleFarmer && role != models.RoleCustomer {
		return ErrForbidden
	}

	hash, err := s.accountRepo.GetPasswordHash(ctx, id)
	if err != nil {
		return fmt.Errorf("deleting account: %w", err)
	}
	if err := credentials.VerifyPassword(password, hash); err != nil {
		if errors.Is(err, credentials.ErrPasswordMismatch) {
			return ErrInvalidCredentials
		}
		return fmt.Errorf("deleting account: verifying password: %w", err)
	}

	// Checked before anything is touched, but not under a lock: a payment
	// that completes between here and DeleteAccount below still slips
	// through. Closing that would mean locking every plot of a farm against
	// the Stripe webhook, which is a lot of machinery for a window this small.
	blockers, err := s.accountRepo.GetDeletionBlockers(ctx, id)
	if err != nil {
		return fmt.Errorf("deleting account: checking blockers: %w", err)
	}
	if blockers.OpenRentals {
		return ErrAccountHasOpenRentals
	}
	if blockers.PendingPayment {
		return ErrAccountHasPendingPayment
	}

	if role == models.RoleFarmer {
		if err := s.cancelSubscription(ctx, id); err != nil {
			return fmt.Errorf("deleting account: %w", err)
		}
	}

	if err := s.accountRepo.DeleteAccount(ctx, id, role); err != nil {
		return fmt.Errorf("deleting account: %w", err)
	}
	return nil
}

// cancelSubscription stops a farmer's billing before the account goes, so a
// deleted farmer is never charged again. It runs before the anonymisation
// rather than after because Stripe cannot take part in the transaction: if
// the database step then fails, the farmer is left with a live account and no
// subscription, which they can fix by subscribing again -- the other order
// could leave a deleted account that keeps paying.
func (s *accountService) cancelSubscription(ctx context.Context, farmer uuid.UUID) error {
	sub, err := s.farmerSubRepo.GetActiveSubscriptionByFarmer(ctx, farmer)
	if errors.Is(err, ErrNotFound) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("looking up subscription: %w", err)
	}
	if sub.StripeSubscriptionID == nil {
		return nil
	}

	if err := s.paymentGateway.CancelSubscription(ctx, *sub.StripeSubscriptionID); err != nil {
		return fmt.Errorf("canceling subscription: %w", err)
	}
	// The customer.subscription.deleted webhook would mark it too, but not
	// before the account is gone; a retried delivery then finds it already
	// canceled, which is the no-op it is guarded to be.
	if _, err := s.farmerSubRepo.CancelSubscription(ctx, *sub.StripeSubscriptionID); err != nil && !errors.Is(err, ErrCheckoutAlreadyProcessed) {
		return fmt.Errorf("marking subscription canceled: %w", err)
	}
	return nil
}

// knownRoles is the closed set a filter may name. models.Role is a string
// type, so without this any typo silently becomes "match nothing".
var knownRoles = []models.Role{models.RoleAdmin, models.RoleFarmer, models.RoleCustomer}
