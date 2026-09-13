package services

import (
	"context"
	"errors"
	"fmt"

	"github.com/Neue-Konzepte-BaaS/backend/internal/models"
	"github.com/google/uuid"
)

type CropService interface {
	// CreateCrop adds a new crop to the catalog. Admin only.
	CreateCrop(ctx context.Context, name string, durationMonths int32) (models.Crop, error)
	// GetAllCrops returns the full crop catalog.
	GetAllCrops(ctx context.Context) ([]models.Crop, error)
	// SetFieldCrops replaces the crops a field offers, after checking the
	// farmer owns that field.
	SetFieldCrops(ctx context.Context, farmer, field uuid.UUID, cropIDs []uuid.UUID) ([]models.Crop, error)
}

type cropService struct {
	fieldRepo FieldRepository
	cropRepo  CropRepository
}

func NewCropService(fieldRepo FieldRepository, cropRepo CropRepository) CropService {
	return &cropService{fieldRepo: fieldRepo, cropRepo: cropRepo}
}

func (s *cropService) CreateCrop(ctx context.Context, name string, durationMonths int32) (models.Crop, error) {
	crop, err := s.cropRepo.CreateCrop(ctx, name, durationMonths)
	if err != nil {
		if errors.Is(err, ErrCropNameTaken) {
			return models.Crop{}, err
		}
		return models.Crop{}, fmt.Errorf("creating crop: %w", err)
	}
	return crop, nil
}

func (s *cropService) GetAllCrops(ctx context.Context) ([]models.Crop, error) {
	crops, err := s.cropRepo.GetAllCrops(ctx)
	if err != nil {
		return nil, fmt.Errorf("getting crops: %w", err)
	}
	return crops, nil
}

func (s *cropService) SetFieldCrops(ctx context.Context, farmer, field uuid.UUID, cropIDs []uuid.UUID) ([]models.Crop, error) {
	owner, err := s.fieldRepo.GetFieldOwner(ctx, field)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			return nil, err
		}
		return nil, fmt.Errorf("looking up field owner: %w", err)
	}
	if owner != farmer {
		return nil, ErrForbidden
	}

	if err := s.cropRepo.SetFieldCrops(ctx, field, cropIDs); err != nil {
		if errors.Is(err, ErrNotFound) {
			return nil, err
		}
		return nil, fmt.Errorf("setting field crops: %w", err)
	}

	crops, err := s.cropRepo.GetCropsByField(ctx, field)
	if err != nil {
		return nil, fmt.Errorf("getting field crops: %w", err)
	}
	return crops, nil
}
