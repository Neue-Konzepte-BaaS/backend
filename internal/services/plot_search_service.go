package services

import (
	"context"
	"errors"
	"fmt"

	"github.com/Neue-Konzepte-BaaS/backend/internal/models"
	"github.com/google/uuid"
)

type PlotSearchService interface {
	// FindNearestByCoordinates finds the plots nearest to the given point,
	// only farm's plots when farm is non-nil.
	FindNearestByCoordinates(ctx context.Context, lon, lat float64, farm *uuid.UUID, limit int32) ([]models.NearbyPlot, error)
	// FindNearestByLocation resolves postalCode/city to coordinates, then
	// finds the plots nearest to that point. Exactly one of
	// postalCode/city should be set.
	FindNearestByLocation(ctx context.Context, postalCode, city string, farm *uuid.UUID, limit int32) ([]models.NearbyPlot, error)
}

type plotSearchService struct {
	plotRepo       PlotRepository
	postalCodeRepo PostalCodeRepository
	cropRepo       CropRepository
	seasonRepo     SeasonRepository
}

func NewPlotSearchService(plotRepo PlotRepository, postalCodeRepo PostalCodeRepository, cropRepo CropRepository, seasonRepo SeasonRepository) PlotSearchService {
	return &plotSearchService{plotRepo: plotRepo, postalCodeRepo: postalCodeRepo, cropRepo: cropRepo, seasonRepo: seasonRepo}
}

func (s *plotSearchService) FindNearestByCoordinates(ctx context.Context, lon, lat float64, farm *uuid.UUID, limit int32) ([]models.NearbyPlot, error) {
	plots, err := s.plotRepo.GetNearestPlots(ctx, lon, lat, farm, limit)
	if err != nil {
		return nil, fmt.Errorf("getting nearest plots: %w", err)
	}

	plotIDs := make([]uuid.UUID, len(plots))
	for i, plot := range plots {
		plotIDs[i] = plot.ID
	}

	offeringsByPlot, err := s.cropRepo.GetPricedCropOfferingsByPlots(ctx, plotIDs)
	if err != nil {
		return nil, fmt.Errorf("getting plot crop offerings: %w", err)
	}

	// One (crop, farm) pair per distinct crop offered on each plot, so a crop
	// offered on several plots of the same farm resolves its season once.
	seen := make(map[models.CropAtFarm]struct{})
	var pairs []models.CropAtFarm
	for _, plot := range plots {
		for _, offering := range offeringsByPlot[plot.ID] {
			key := models.CropAtFarm{Crop: offering.ID, Farm: plot.Farm}
			if _, ok := seen[key]; ok {
				continue
			}
			seen[key] = struct{}{}
			pairs = append(pairs, key)
		}
	}
	seasonsByCropAndFarm, err := s.seasonRepo.GetEffectiveSeasonsForCrops(ctx, pairs)
	if err != nil {
		return nil, fmt.Errorf("getting effective seasons: %w", err)
	}

	for i, plot := range plots {
		offerings := offeringsByPlot[plot.ID]
		withSeasons := make([]models.PlotCropOffering, len(offerings))
		for j, offering := range offerings {
			withSeasons[j] = offering
			if season, ok := seasonsByCropAndFarm[models.CropAtFarm{Crop: offering.ID, Farm: plot.Farm}]; ok {
				withSeasons[j].Season = &season
			}
		}
		plots[i].Crops = withSeasons
	}
	return plots, nil
}

func (s *plotSearchService) FindNearestByLocation(ctx context.Context, postalCode, city string, farm *uuid.UUID, limit int32) ([]models.NearbyPlot, error) {
	lon, lat, err := s.postalCodeRepo.FindCoordinates(ctx, postalCode, city)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			return nil, err
		}
		return nil, fmt.Errorf("resolving location: %w", err)
	}

	return s.FindNearestByCoordinates(ctx, lon, lat, farm, limit)
}
