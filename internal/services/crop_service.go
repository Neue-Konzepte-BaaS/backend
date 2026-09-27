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
	// DeleteCrop removes a crop from the catalog. Returns ErrConflict if the
	// crop is referenced by a rental, ErrNotFound if the id is unknown.
	DeleteCrop(ctx context.Context, id uuid.UUID) error
	// GetAllCrops returns the full crop catalog.
	GetAllCrops(ctx context.Context) ([]models.Crop, error)
	// SetPlotCrops replaces the plot's base price and the crops it offers,
	// after checking the farmer owns the field that plot belongs to.
	SetPlotCrops(ctx context.Context, farmer, plot uuid.UUID, basePriceCentsPerSqmPerWeek int32, cropIDs []uuid.UUID) (models.PlotWithCrops, error)
}

type cropService struct {
	farmRepo  FarmRepository
	fieldRepo FieldRepository
	plotRepo  PlotRepository
	cropRepo  CropRepository
}

func NewCropService(farmRepo FarmRepository, fieldRepo FieldRepository, plotRepo PlotRepository, cropRepo CropRepository) CropService {
	return &cropService{farmRepo: farmRepo, fieldRepo: fieldRepo, plotRepo: plotRepo, cropRepo: cropRepo}
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

func (s *cropService) DeleteCrop(ctx context.Context, id uuid.UUID) error {
	if err := s.cropRepo.DeleteCrop(ctx, id); err != nil {
		if errors.Is(err, ErrConflict) {
			return err
		}
		return fmt.Errorf("deleting crop: %w", err)
	}
	return nil
}

func (s *cropService) GetAllCrops(ctx context.Context) ([]models.Crop, error) {
	crops, err := s.cropRepo.GetAllCrops(ctx)
	if err != nil {
		return nil, fmt.Errorf("getting crops: %w", err)
	}
	return crops, nil
}

func (s *cropService) SetPlotCrops(ctx context.Context, farmer, plot uuid.UUID, basePriceCentsPerSqmPerWeek int32, cropIDs []uuid.UUID) (models.PlotWithCrops, error) {
	callerFarm, err := s.farmRepo.GetFarmIDByFarmerID(ctx, farmer)
	if err != nil {
		return models.PlotWithCrops{}, fmt.Errorf("looking up farm: %w", err)
	}

	existingPlot, err := s.plotRepo.GetPlotByID(ctx, plot)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			return models.PlotWithCrops{}, err
		}
		return models.PlotWithCrops{}, fmt.Errorf("looking up plot: %w", err)
	}

	fieldFarm, err := s.fieldRepo.GetFieldFarm(ctx, existingPlot.Field)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			return models.PlotWithCrops{}, err
		}
		return models.PlotWithCrops{}, fmt.Errorf("looking up field farm: %w", err)
	}
	if fieldFarm != callerFarm {
		return models.PlotWithCrops{}, ErrForbidden
	}

	if err := s.cropRepo.SetPlotCrops(ctx, plot, basePriceCentsPerSqmPerWeek, cropIDs); err != nil {
		if errors.Is(err, ErrNotFound) {
			return models.PlotWithCrops{}, err
		}
		return models.PlotWithCrops{}, fmt.Errorf("setting plot crops: %w", err)
	}

	crops, err := s.cropRepo.GetCropsByPlot(ctx, plot)
	if err != nil {
		return models.PlotWithCrops{}, fmt.Errorf("getting plot crops: %w", err)
	}

	existingPlot.BasePriceCentsPerSqmPerWeek = &basePriceCentsPerSqmPerWeek
	return models.PlotWithCrops{Plot: existingPlot, Crops: crops}, nil
}
