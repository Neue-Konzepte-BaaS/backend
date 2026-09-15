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
	// GetFieldsWithPlots returns all fields owned by the farmer, along with
	// the plots belonging to each of those fields.
	GetFieldsWithPlots(ctx context.Context, farmer uuid.UUID) ([]models.FieldWithPlots, error)
}

type PlotService interface {
	// CreatePlot creates a plot on the given field, after checking that
	// farmer owns that field.
	CreatePlot(ctx context.Context, farmer uuid.UUID, field uuid.UUID, name string, coordinates *geom.Polygon) (models.Plot, error)
}

type fieldService struct {
	fieldRepo FieldRepository
	plotRepo  PlotRepository
	cropRepo  CropRepository
}

func NewFieldService(fieldRepo FieldRepository, plotRepo PlotRepository, cropRepo CropRepository) FieldService {
	return &fieldService{fieldRepo: fieldRepo, plotRepo: plotRepo, cropRepo: cropRepo}
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

func (s *fieldService) GetFieldsWithPlots(ctx context.Context, farmer uuid.UUID) ([]models.FieldWithPlots, error) {
	fields, err := s.fieldRepo.GetFieldsByFarmer(ctx, farmer)
	if err != nil {
		return nil, fmt.Errorf("getting fields: %w", err)
	}

	fieldIDs := make([]uuid.UUID, len(fields))
	for i, field := range fields {
		fieldIDs[i] = field.ID
	}

	plots, err := s.plotRepo.GetPlotsByFields(ctx, fieldIDs)
	if err != nil {
		return nil, fmt.Errorf("getting plots: %w", err)
	}

	plotIDs := make([]uuid.UUID, len(plots))
	for i, plot := range plots {
		plotIDs[i] = plot.ID
	}

	cropsByPlot, err := s.cropRepo.GetCropsByPlots(ctx, plotIDs)
	if err != nil {
		return nil, fmt.Errorf("getting plot crops: %w", err)
	}

	plotsByField := make(map[uuid.UUID][]models.PlotWithCrops, len(fields))
	for _, plot := range plots {
		plotsByField[plot.Field] = append(plotsByField[plot.Field], models.PlotWithCrops{
			Plot:  plot,
			Crops: cropsByPlot[plot.ID],
		})
	}

	result := make([]models.FieldWithPlots, len(fields))
	for i, field := range fields {
		result[i] = models.FieldWithPlots{
			Field: field,
			Plots: plotsByField[field.ID],
		}
	}
	return result, nil
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

	created, err := s.plotRepo.CreatePlot(ctx, plot)
	if err != nil {
		if errors.Is(err, ErrInvalidGeometry) {
			return models.Plot{}, err
		}
		return models.Plot{}, fmt.Errorf("creating plot: %w", err)
	}

	return created, nil
}
