package services

import (
	"context"
	"errors"
	"fmt"

	"github.com/Neue-Konzepte-BaaS/backend/internal/models"
	"github.com/google/uuid"
)

type RentalService interface {
	// RentPlot books the plot for the customer with the chosen crop, starting
	// now and running for that crop's duration. Returns ErrCropNotOffered if
	// the plot does not offer that crop, ErrPlotUnavailable if the plot is
	// already rented for part of that period, and ErrNotFound if the plot or
	// crop does not exist.
	RentPlot(ctx context.Context, customer, plot, crop uuid.UUID) (models.Rental, error)
	// GetRentals returns the customer's own rentals, newest first.
	GetRentals(ctx context.Context, customer uuid.UUID) ([]models.RentalWithPlot, error)
	// GetRentalsForFarmer returns every rental on the farmer's own plots,
	// active and historic, newest first.
	GetRentalsForFarmer(ctx context.Context, farmer uuid.UUID) ([]models.RentalWithPlotAndCustomer, error)
}

type rentalService struct {
	rentalRepo RentalRepository
	plotRepo   PlotRepository
	cropRepo   CropRepository
}

func NewRentalService(rentalRepo RentalRepository, plotRepo PlotRepository, cropRepo CropRepository) RentalService {
	return &rentalService{rentalRepo: rentalRepo, plotRepo: plotRepo, cropRepo: cropRepo}
}

func (s *rentalService) RentPlot(ctx context.Context, customer, plot, crop uuid.UUID) (models.Rental, error) {
	cropDetails, err := s.cropRepo.GetCropByID(ctx, crop)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			return models.Rental{}, err
		}
		return models.Rental{}, fmt.Errorf("getting crop: %w", err)
	}

	// A plot lookup also validates the plot exists: GetCropsByPlot alone
	// would silently return no rows for a bad plot id, which would otherwise
	// misreport a 404 as ErrCropNotOffered below.
	if _, err := s.plotRepo.GetPlotField(ctx, plot); err != nil {
		if errors.Is(err, ErrNotFound) {
			return models.Rental{}, err
		}
		return models.Rental{}, fmt.Errorf("checking plot exists: %w", err)
	}

	offeredCrops, err := s.cropRepo.GetCropsByPlot(ctx, plot)
	if err != nil {
		return models.Rental{}, fmt.Errorf("getting plot crops: %w", err)
	}
	if !cropOffered(offeredCrops, crop) {
		return models.Rental{}, ErrCropNotOffered
	}

	rental, err := s.rentalRepo.CreateRental(ctx, plot, customer, crop, cropDetails.DurationMonths)
	if err != nil {
		if errors.Is(err, ErrPlotUnavailable) || errors.Is(err, ErrNotFound) {
			return models.Rental{}, err
		}
		return models.Rental{}, fmt.Errorf("creating rental: %w", err)
	}
	return rental, nil
}

func cropOffered(crops []models.Crop, crop uuid.UUID) bool {
	for _, c := range crops {
		if c.ID == crop {
			return true
		}
	}
	return false
}

func (s *rentalService) GetRentals(ctx context.Context, customer uuid.UUID) ([]models.RentalWithPlot, error) {
	rentals, err := s.rentalRepo.GetRentalsByCustomer(ctx, customer)
	if err != nil {
		return nil, fmt.Errorf("getting rentals: %w", err)
	}
	return rentals, nil
}

func (s *rentalService) GetRentalsForFarmer(ctx context.Context, farmer uuid.UUID) ([]models.RentalWithPlotAndCustomer, error) {
	rentals, err := s.rentalRepo.GetRentalsByFarmer(ctx, farmer)
	if err != nil {
		return nil, fmt.Errorf("getting farmer rentals: %w", err)
	}
	return rentals, nil
}
