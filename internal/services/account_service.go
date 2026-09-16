package services

import (
	"context"
	"fmt"
	"slices"
	"strings"

	"github.com/Neue-Konzepte-BaaS/backend/internal/models"
)

type AccountService interface {
	// ListAccounts returns one page of the platform's accounts. Only an admin
	// may call it; every other role gets ErrForbidden. A Role filter that is
	// not one of the three known roles is ErrInvalidFilter.
	ListAccounts(ctx context.Context, role models.Role, filter models.AccountListFilter) (models.Page[models.AccountListing], error)
}

type accountService struct {
	accountRepo AccountRepository
}

func NewAccountService(accountRepo AccountRepository) AccountService {
	return &accountService{accountRepo: accountRepo}
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
