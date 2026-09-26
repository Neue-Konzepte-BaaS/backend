package services

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/Neue-Konzepte-BaaS/backend/internal/models"
	"github.com/google/uuid"
)

type FarmService interface {
	// GetFarm returns public details of the farm with the given id.
	GetFarm(ctx context.Context, farmID uuid.UUID) (models.Farm, error)
	// GetMyFarm returns the farmer's own farm. Returns ErrNotFound if the
	// account owns no farm.
	GetMyFarm(ctx context.Context, farmer uuid.UUID) (models.Farm, error)
	// UpdateMyFarm overwrites the editable fields of the farmer's own farm and
	// returns it as GetMyFarm would. The farm is found by its owner, never by
	// an id the caller names. Returns ErrNotFound if the account owns no farm.
	UpdateMyFarm(ctx context.Context, farmer uuid.UUID, update models.FarmUpdate) (models.Farm, error)
	// ListFarms returns one page of the platform's farms. Only an admin may
	// call it; every other role gets ErrForbidden. Unlike GetFarm, which is
	// public, this is a back-office view: it carries the owner and the
	// holdings, so it is gated.
	ListFarms(ctx context.Context, role models.Role, filter models.FarmListFilter) (models.Page[models.FarmListing], error)
}

type farmService struct {
	farmRepo FarmRepository
}

func NewFarmService(farmRepo FarmRepository) FarmService {
	return &farmService{farmRepo: farmRepo}
}

func (s *farmService) GetFarm(ctx context.Context, farmID uuid.UUID) (models.Farm, error) {
	farm, err := s.farmRepo.GetFarmByID(ctx, farmID)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			return models.Farm{}, err
		}
		return models.Farm{}, fmt.Errorf("getting farm: %w", err)
	}
	return farm, nil
}

func (s *farmService) GetMyFarm(ctx context.Context, farmer uuid.UUID) (models.Farm, error) {
	farmID, err := s.farmRepo.GetFarmIDByFarmerID(ctx, farmer)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			return models.Farm{}, err
		}
		return models.Farm{}, fmt.Errorf("looking up farm: %w", err)
	}
	return s.GetFarm(ctx, farmID)
}

func (s *farmService) UpdateMyFarm(ctx context.Context, farmer uuid.UUID, update models.FarmUpdate) (models.Farm, error) {
	farmID, err := s.farmRepo.UpdateFarmByFarmer(ctx, farmer, update)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			return models.Farm{}, err
		}
		return models.Farm{}, fmt.Errorf("updating farm: %w", err)
	}
	// Read back rather than echo the input: the response carries the derived
	// area too, and is then exactly what GET /api/farms/me answers.
	return s.GetFarm(ctx, farmID)
}

func (s *farmService) ListFarms(ctx context.Context, role models.Role, filter models.FarmListFilter) (models.Page[models.FarmListing], error) {
	// The route is gated by RequireRole(admin), so this is defence in depth --
	// the same belt-and-braces as statisticsService.GetStatistics. The scope is
	// the caller's role and nothing else: there is no parameter that widens it.
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
