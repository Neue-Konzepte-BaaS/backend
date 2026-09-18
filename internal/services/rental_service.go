package services

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/Neue-Konzepte-BaaS/backend/internal/models"
	"github.com/google/uuid"
)

// minRentalNotice and maxRentalNotice bound how far out a customer can pick
// a rental's start date: at least a day out, since the farmer needs time to
// review the request, and no more than 60 days out, since the platform does
// not want to hold a plot against a request that far in advance.
const (
	minRentalNotice = 24 * time.Hour
	maxRentalNotice = 60 * 24 * time.Hour
)

type RentalService interface {
	// RequestRental records the customer's request to book the plot with the
	// chosen crop, starting at startAt and running for that crop's duration.
	// startAt must be between 1 and 60 days from now. Returns
	// ErrInvalidRentalRequest if startAt is out of that window or message is
	// blank, ErrCropNotOffered if the plot does not offer that crop,
	// ErrPlotUnavailable if the plot is already requested or rented for part
	// of that period, and ErrNotFound if the plot or crop does not exist.
	RequestRental(ctx context.Context, customer, plot, crop uuid.UUID, startAt time.Time, message string) (models.Rental, error)
	// ApproveRental approves a still-requested rental on one of the farmer's
	// own plots. Returns ErrForbidden if the farmer does not own the plot,
	// and ErrRentalAlreadyDecided if the rental is not in the Requested
	// state (including if the id does not exist).
	ApproveRental(ctx context.Context, farmer, rental uuid.UUID) (models.Rental, error)
	// DeclineRental declines a still-requested rental on one of the farmer's
	// own plots, freeing the plot for that period. Returns ErrForbidden if
	// the farmer does not own the plot, and ErrRentalAlreadyDecided if the
	// rental is not in the Requested state (including if the id does not
	// exist).
	DeclineRental(ctx context.Context, farmer, rental uuid.UUID) (models.Rental, error)
	// GetRentals returns the customer's own rentals, newest first.
	GetRentals(ctx context.Context, customer uuid.UUID) ([]models.RentalWithPlot, error)
	// GetRentalsForFarmer returns every rental on the farmer's own plots,
	// active and historic, newest first.
	GetRentalsForFarmer(ctx context.Context, farmer uuid.UUID) ([]models.RentalWithPlotAndCustomer, error)
}

type rentalService struct {
	farmRepo   FarmRepository
	fieldRepo  FieldRepository
	rentalRepo RentalRepository
	plotRepo   PlotRepository
	cropRepo   CropRepository
}

func NewRentalService(farmRepo FarmRepository, fieldRepo FieldRepository, rentalRepo RentalRepository, plotRepo PlotRepository, cropRepo CropRepository) RentalService {
	return &rentalService{farmRepo: farmRepo, fieldRepo: fieldRepo, rentalRepo: rentalRepo, plotRepo: plotRepo, cropRepo: cropRepo}
}

func (s *rentalService) RequestRental(ctx context.Context, customer, plot, crop uuid.UUID, startAt time.Time, message string) (models.Rental, error) {
	if strings.TrimSpace(message) == "" {
		return models.Rental{}, ErrInvalidRentalRequest
	}
	notice := time.Until(startAt)
	if notice < minRentalNotice || notice > maxRentalNotice {
		return models.Rental{}, ErrInvalidRentalRequest
	}

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

	rental, err := s.rentalRepo.CreateRentalRequest(ctx, plot, customer, crop, startAt, cropDetails.DurationMonths, message)
	if err != nil {
		if errors.Is(err, ErrPlotUnavailable) || errors.Is(err, ErrNotFound) {
			return models.Rental{}, err
		}
		return models.Rental{}, fmt.Errorf("creating rental: %w", err)
	}
	return rental, nil
}

func (s *rentalService) ApproveRental(ctx context.Context, farmer, rental uuid.UUID) (models.Rental, error) {
	return s.decideRental(ctx, farmer, rental, models.RentalStatusApproved)
}

func (s *rentalService) DeclineRental(ctx context.Context, farmer, rental uuid.UUID) (models.Rental, error) {
	return s.decideRental(ctx, farmer, rental, models.RentalStatusDeclined)
}

func (s *rentalService) decideRental(ctx context.Context, farmer, rentalID uuid.UUID, status models.RentalStatus) (models.Rental, error) {
	existing, err := s.rentalRepo.GetRentalWithFieldByID(ctx, rentalID)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			return models.Rental{}, err
		}
		return models.Rental{}, fmt.Errorf("looking up rental: %w", err)
	}

	callerFarm, err := s.farmRepo.GetFarmIDByFarmerID(ctx, farmer)
	if err != nil {
		return models.Rental{}, fmt.Errorf("looking up farm: %w", err)
	}
	fieldFarm, err := s.fieldRepo.GetFieldFarm(ctx, existing.Field)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			return models.Rental{}, err
		}
		return models.Rental{}, fmt.Errorf("looking up field farm: %w", err)
	}
	if fieldFarm != callerFarm {
		return models.Rental{}, ErrForbidden
	}

	rental, err := s.rentalRepo.UpdateRentalStatus(ctx, rentalID, status)
	if err != nil {
		if errors.Is(err, ErrRentalAlreadyDecided) {
			return models.Rental{}, err
		}
		return models.Rental{}, fmt.Errorf("updating rental status: %w", err)
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
	farmID, err := s.farmRepo.GetFarmIDByFarmerID(ctx, farmer)
	if err != nil {
		return nil, fmt.Errorf("looking up farm: %w", err)
	}
	rentals, err := s.rentalRepo.GetRentalsByFarm(ctx, farmID)
	if err != nil {
		return nil, fmt.Errorf("getting farmer rentals: %w", err)
	}
	return rentals, nil
}
