package services

import (
	"context"
	"errors"
	"fmt"

	"github.com/Neue-Konzepte-BaaS/backend/internal/models"
	"github.com/google/uuid"
	geom "github.com/twpayne/go-geom"
)

type FieldService interface {
	CreateField(ctx context.Context, farmer uuid.UUID, name string, coordinates *geom.Polygon) (models.Field, error)
}

type PlotService interface {
	// CreatePlot creates a plot on the given field, after checking that
	// farmer owns that field.
	CreatePlot(ctx context.Context, farmer uuid.UUID, field uuid.UUID, name string, coordinates *geom.Polygon) (models.Plot, error)
}

type fieldService struct {
	fieldRepo FieldRepository
}

func NewFieldService(fieldRepo FieldRepository) FieldService {
	return &fieldService{fieldRepo: fieldRepo}
}

func (s *fieldService) CreateField(ctx context.Context, farmer uuid.UUID, name string, coordinates *geom.Polygon) (models.Field, error) {
	field := models.Field{
		Name:        name,
		Farmer:      farmer,
		Coordinates: coordinates,
	}

	id, err := s.fieldRepo.CreateField(ctx, field)
	if err != nil {
		if errors.Is(err, ErrInvalidGeometry) {
			return models.Field{}, err
		}
		return models.Field{}, fmt.Errorf("creating field: %w", err)
	}

	field.ID = id
	return field, nil
}

type plotService struct {
	fieldRepo FieldRepository
	plotRepo  PlotRepository
}

func NewPlotService(fieldRepo FieldRepository, plotRepo PlotRepository) PlotService {
	return &plotService{fieldRepo: fieldRepo, plotRepo: plotRepo}
}

func (s *plotService) CreatePlot(ctx context.Context, farmer uuid.UUID, fieldID uuid.UUID, name string, coordinates *geom.Polygon) (models.Plot, error) {
	owner, err := s.fieldRepo.GetFieldOwner(ctx, fieldID)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			return models.Plot{}, err
		}
		return models.Plot{}, fmt.Errorf("looking up field owner: %w", err)
	}
	if owner != farmer {
		return models.Plot{}, ErrForbidden
	}

	plot := models.Plot{
		Name:        name,
		Field:       fieldID,
		Coordinates: coordinates,
	}

	id, err := s.plotRepo.CreatePlot(ctx, plot)
	if err != nil {
		if errors.Is(err, ErrInvalidGeometry) {
			return models.Plot{}, err
		}
		return models.Plot{}, fmt.Errorf("creating plot: %w", err)
	}

	plot.ID = id
	return plot, nil
}
