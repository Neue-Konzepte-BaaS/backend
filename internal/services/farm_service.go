package services

import (
	"context"
	"errors"
	"fmt"

	"github.com/Neue-Konzepte-BaaS/backend/internal/models"
	"github.com/google/uuid"
)

type FarmService interface {
	// GetFarm returns public details of the farm with the given id.
	GetFarm(ctx context.Context, farmID uuid.UUID) (models.Farm, error)
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
