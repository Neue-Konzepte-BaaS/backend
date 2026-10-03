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

type AccountService interface {
	// ListAccounts returns one page of the platform's accounts. Only an admin
	// may call it; every other role gets ErrForbidden. A Role filter that is
	// not one of the three known roles is ErrInvalidFilter.
	ListAccounts(ctx context.Context, role models.Role, filter models.AccountListFilter) (models.Page[models.AccountListing], error)
	// DeleteMyAccount soft-deletes the calling account: scrubs its personal
	// data and marks it so it can no longer log in, be messaged, or (for a
	// farmer) offer plots / show up in farm search. A farmer's active Stripe
	// subscription, if any, is cancelled automatically first. Returns
	// ErrAccountHasActiveRentals if the account (customer) or any of its
	// plots (farmer) has a rental covering right now, and ErrForbidden for
	// any role other than farmer or customer.
	DeleteMyAccount(ctx context.Context, accountID uuid.UUID, role models.Role) error
	// GetMyNotificationPreferences returns the calling customer's email
	// notification preference. Returns ErrForbidden for any role other than
	// customer.
	GetMyNotificationPreferences(ctx context.Context, accountID uuid.UUID, role models.Role) (models.CustomerNotificationPreferences, error)
	// UpdateMyNotificationPreferences overwrites the calling customer's email
	// notification preference. Returns ErrForbidden for any role other than
	// customer.
	UpdateMyNotificationPreferences(ctx context.Context, accountID uuid.UUID, role models.Role, prefs models.CustomerNotificationPreferences) (models.CustomerNotificationPreferences, error)
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

// knownRoles is the closed set a filter may name. models.Role is a string
// type, so without this any typo silently becomes "match nothing".
var knownRoles = []models.Role{models.RoleAdmin, models.RoleFarmer, models.RoleCustomer}

// scrubbedPasswordHash replaces a deleted account's password hash so login
// can never succeed even if the deleted_at check were ever bypassed --
// defense in depth alongside that check. Computed once like auth_service's
// own dummyHash, from a throwaway value nobody is ever told.
var scrubbedPasswordHash, _ = credentials.HashPassword("account-deleted-" + uuid.NewString())

func (s *accountService) DeleteMyAccount(ctx context.Context, accountID uuid.UUID, role models.Role) error {
	switch role {
	case models.RoleCustomer:
		active, err := s.accountRepo.HasActiveRentalAsCustomer(ctx, accountID)
		if err != nil {
			return fmt.Errorf("checking active rentals: %w", err)
		}
		if active {
			return ErrAccountHasActiveRentals
		}

	case models.RoleFarmer:
		active, err := s.accountRepo.HasActiveRentalAsFarmer(ctx, accountID)
		if err != nil {
			return fmt.Errorf("checking active rentals: %w", err)
		}
		if active {
			return ErrAccountHasActiveRentals
		}

		if err := s.cancelFarmerSubscription(ctx, accountID); err != nil {
			return err
		}

	default:
		// Admins are out of scope for self-service deletion.
		return ErrForbidden
	}

	scrubbedEmail := "deleted+" + accountID.String() + "@deleted.invalid"
	if err := s.accountRepo.SoftDeleteAccount(ctx, accountID, "Deleted", "User", scrubbedEmail, scrubbedPasswordHash); err != nil {
		return fmt.Errorf("deleting account: %w", err)
	}
	return nil
}

func (s *accountService) GetMyNotificationPreferences(ctx context.Context, accountID uuid.UUID, role models.Role) (models.CustomerNotificationPreferences, error) {
	if role != models.RoleCustomer {
		return models.CustomerNotificationPreferences{}, ErrForbidden
	}

	prefs, err := s.accountRepo.GetCustomerNotificationPreferences(ctx, accountID)
	if err != nil {
		return models.CustomerNotificationPreferences{}, fmt.Errorf("getting notification preferences: %w", err)
	}
	return prefs, nil
}

func (s *accountService) UpdateMyNotificationPreferences(ctx context.Context, accountID uuid.UUID, role models.Role, prefs models.CustomerNotificationPreferences) (models.CustomerNotificationPreferences, error) {
	if role != models.RoleCustomer {
		return models.CustomerNotificationPreferences{}, ErrForbidden
	}

	if err := s.accountRepo.UpdateCustomerNotificationPreferences(ctx, accountID, prefs); err != nil {
		return models.CustomerNotificationPreferences{}, fmt.Errorf("updating notification preferences: %w", err)
	}
	return prefs, nil
}

// cancelFarmerSubscription cancels the farmer's active subscription, if any,
// both on Stripe and in the local ledger. Called before the account is
// scrubbed: canceling first means a failure here leaves the farmer with a
// working, still-subscribed account they can retry deleting, rather than a
// login-locked account with a subscription still silently charging them.
func (s *accountService) cancelFarmerSubscription(ctx context.Context, farmerID uuid.UUID) error {
	sub, err := s.farmerSubRepo.GetActiveSubscriptionByFarmer(ctx, farmerID)
	if errors.Is(err, ErrNotFound) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("checking subscription: %w", err)
	}
	if sub.StripeSubscriptionID == nil {
		return nil
	}

	if err := s.paymentGateway.CancelSubscription(ctx, *sub.StripeSubscriptionID); err != nil {
		return fmt.Errorf("canceling stripe subscription: %w", err)
	}
	if _, err := s.farmerSubRepo.CancelSubscription(ctx, *sub.StripeSubscriptionID); err != nil {
		return fmt.Errorf("canceling subscription record: %w", err)
	}
	return nil
}
