package services

import (
	"context"
	"errors"
	"fmt"

	"github.com/Neue-Konzepte-BaaS/backend/internal/models"
	"github.com/google/uuid"
)

// RentalDurationMonths is how long a plot is rented for. Fixed for now; the
// rental period is stored as a range so making this caller-supplied later
// needs no schema change.
const RentalDurationMonths = 6

type RentalService interface {
	// RentPlot books the plot for the customer, starting now and running for
	// RentalDurationMonths. Returns ErrPlotUnavailable if the plot is already
	// rented for part of that period, and ErrNotFound if it does not exist.
	RentPlot(ctx context.Context, customer, plot uuid.UUID) (models.Rental, error)
	// GetRentals returns the customer's own rentals, newest first.
	GetRentals(ctx context.Context, customer uuid.UUID) ([]models.RentalWithPlot, error)
}

type rentalService struct {
	rentalRepo RentalRepository
}

func NewRentalService(rentalRepo RentalRepository) RentalService {
	return &rentalService{rentalRepo: rentalRepo}
}

func (s *rentalService) RentPlot(ctx context.Context, customer, plot uuid.UUID) (models.Rental, error) {
	rental, err := s.rentalRepo.CreateRental(ctx, plot, customer, RentalDurationMonths)
	if err != nil {
		if errors.Is(err, ErrPlotUnavailable) || errors.Is(err, ErrNotFound) {
			return models.Rental{}, err
		}
		return models.Rental{}, fmt.Errorf("creating rental: %w", err)
	}
	return rental, nil
}

func (s *rentalService) GetRentals(ctx context.Context, customer uuid.UUID) ([]models.RentalWithPlot, error) {
	rentals, err := s.rentalRepo.GetRentalsByCustomer(ctx, customer)
	if err != nil {
		return nil, fmt.Errorf("getting rentals: %w", err)
	}
	return rentals, nil
}
