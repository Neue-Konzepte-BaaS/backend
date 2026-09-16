package services

import (
	"context"
	"fmt"
	"strings"

	"github.com/Neue-Konzepte-BaaS/backend/internal/models"
)

type FarmService interface {
	// ListFarms returns one page of the platform's farms. Only an admin may
	// call it; every other role gets ErrForbidden.
	ListFarms(ctx context.Context, role models.Role, filter models.FarmListFilter) (models.Page[models.FarmListing], error)
}

type farmService struct {
	farmRepo FarmRepository
}

func NewFarmService(farmRepo FarmRepository) FarmService {
	return &farmService{farmRepo: farmRepo}
}

func (s *farmService) ListFarms(ctx context.Context, role models.Role, filter models.FarmListFilter) (models.Page[models.FarmListing], error) {
	// Defence in depth behind RequireRole(admin); see accountService.
	if role != models.RoleAdmin {
		return models.Page[models.FarmListing]{}, ErrForbidden
	}

	filter.Query = strings.TrimSpace(filter.Query)
	filter.Limit, filter.Offset = clampPagination(filter.Limit, filter.Offset)

	page, err := s.farmRepo.ListFarms(ctx, filter)
	if err != nil {
		return models.Page[models.FarmListing]{}, fmt.Errorf("listing farms: %w", err)
	}

	// Same derivation the statistics endpoint applies, so a farm's occupancy
	// reads identically on both screens.
	for i := range page.Items {
		page.Items[i].Plots = derivePlotFigures(page.Items[i].Plots)
	}
	return page, nil
}
