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
	// SetPlotCrops replaces the crops a plot offers, after checking the
	// farmer owns the field that plot belongs to.
	SetPlotCrops(ctx context.Context, farmer, plot uuid.UUID, cropIDs []uuid.UUID) ([]models.Crop, error)
}

type cropService struct {
	fieldRepo FieldRepository
	plotRepo  PlotRepository
	cropRepo  CropRepository
}

func NewCropService(fieldRepo FieldRepository, plotRepo PlotRepository, cropRepo CropRepository) CropService {
	return &cropService{fieldRepo: fieldRepo, plotRepo: plotRepo, cropRepo: cropRepo}
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

func (s *cropService) SetPlotCrops(ctx context.Context, farmer, plot uuid.UUID, cropIDs []uuid.UUID) ([]models.Crop, error) {
	field, err := s.plotRepo.GetPlotField(ctx, plot)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			return nil, err
		}
		return nil, fmt.Errorf("looking up plot field: %w", err)
	}

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

	if err := s.cropRepo.SetPlotCrops(ctx, plot, cropIDs); err != nil {
		if errors.Is(err, ErrNotFound) {
			return nil, err
		}
		return nil, fmt.Errorf("setting plot crops: %w", err)
	}

	crops, err := s.cropRepo.GetCropsByPlot(ctx, plot)
	if err != nil {
		return nil, fmt.Errorf("getting plot crops: %w", err)
	}
	return crops, nil
}
