package services

import (
	"context"
	"errors"
	"fmt"

	"github.com/Neue-Konzepte-BaaS/backend/internal/models"
)

type PlotSearchService interface {
	// FindNearestByCoordinates finds the plots nearest to the given point.
	FindNearestByCoordinates(ctx context.Context, lon, lat float64, limit int32) ([]models.NearbyPlot, error)
	// FindNearestByLocation resolves postalCode/city to coordinates, then
	// finds the plots nearest to that point. Exactly one of
	// postalCode/city should be set.
	FindNearestByLocation(ctx context.Context, postalCode, city string, limit int32) ([]models.NearbyPlot, error)
}

type plotSearchService struct {
	plotRepo       PlotRepository
	postalCodeRepo PostalCodeRepository
}

func NewPlotSearchService(plotRepo PlotRepository, postalCodeRepo PostalCodeRepository) PlotSearchService {
	return &plotSearchService{plotRepo: plotRepo, postalCodeRepo: postalCodeRepo}
}

func (s *plotSearchService) FindNearestByCoordinates(ctx context.Context, lon, lat float64, limit int32) ([]models.NearbyPlot, error) {
	plots, err := s.plotRepo.GetNearestPlots(ctx, lon, lat, limit)
	if err != nil {
		return nil, fmt.Errorf("getting nearest plots: %w", err)
	}
	return plots, nil
}

func (s *plotSearchService) FindNearestByLocation(ctx context.Context, postalCode, city string, limit int32) ([]models.NearbyPlot, error) {
	lon, lat, err := s.postalCodeRepo.FindCoordinates(ctx, postalCode, city)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			return nil, err
		}
		return nil, fmt.Errorf("resolving location: %w", err)
	}

	return s.FindNearestByCoordinates(ctx, lon, lat, limit)
}
