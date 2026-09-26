package services

import (
	"context"
	"errors"
	"fmt"

	"github.com/Neue-Konzepte-BaaS/backend/internal/models"
	"github.com/google/uuid"
)

// CareGuideService is the weekly care guide: per-crop instructions an admin
// writes once, read back by a tenant against the week their own rental is in.
//
// Authoring is admin-only, like the crop catalog the guide hangs off — a
// farmer offering tomatoes should not have to write tomato advice, and two
// farmers offering the same crop should not have to disagree about it. What is
// farm-specific ("the water is off on Tuesday") is an announcement, which the
// farmer already owns.
type CareGuideService interface {
	// CreateCareInstruction adds one task to a crop's guide. Returns
	// ErrNotFound if the crop does not exist.
	CreateCareInstruction(ctx context.Context, crop uuid.UUID, week int32, title, body string) (models.CareInstruction, error)
	// UpdateCareInstruction rewrites an existing task. Returns ErrNotFound if
	// no instruction has that id.
	UpdateCareInstruction(ctx context.Context, id uuid.UUID, week int32, title, body string) (models.CareInstruction, error)
	// DeleteCareInstruction removes one task. Returns ErrNotFound if no
	// instruction has that id.
	DeleteCareInstruction(ctx context.Context, id uuid.UUID) error
	// GetCareInstructionsForCrop returns one crop's whole guide, in week
	// order — the authoring view, not scoped to any rental.
	GetCareInstructionsForCrop(ctx context.Context, crop uuid.UUID) ([]models.CareInstruction, error)
	// GetCareGuideForCustomer returns one guide per plot the customer is
	// renting right now. A plot whose crop has no guide yet is still
	// returned, with an empty instruction list: the tenant's own week and
	// rental period are worth showing even before anyone writes the advice.
	GetCareGuideForCustomer(ctx context.Context, customer uuid.UUID) ([]models.PlotCareGuide, error)
}

type careGuideService struct {
	careInstructionRepo CareInstructionRepository
	rentalRepo          RentalRepository
}

func NewCareGuideService(careInstructionRepo CareInstructionRepository, rentalRepo RentalRepository) CareGuideService {
	return &careGuideService{careInstructionRepo: careInstructionRepo, rentalRepo: rentalRepo}
}

func (s *careGuideService) CreateCareInstruction(ctx context.Context, crop uuid.UUID, week int32, title, body string) (models.CareInstruction, error) {
	instruction, err := s.careInstructionRepo.CreateCareInstruction(ctx, crop, week, title, body)
	if err != nil {
		if errors.Is(err, ErrNotFound) || errors.Is(err, ErrInvalidCareInstruction) {
			return models.CareInstruction{}, err
		}
		return models.CareInstruction{}, fmt.Errorf("creating care instruction: %w", err)
	}
	return instruction, nil
}

func (s *careGuideService) UpdateCareInstruction(ctx context.Context, id uuid.UUID, week int32, title, body string) (models.CareInstruction, error) {
	instruction, err := s.careInstructionRepo.UpdateCareInstruction(ctx, id, week, title, body)
	if err != nil {
		if errors.Is(err, ErrNotFound) || errors.Is(err, ErrInvalidCareInstruction) {
			return models.CareInstruction{}, err
		}
		return models.CareInstruction{}, fmt.Errorf("updating care instruction: %w", err)
	}
	return instruction, nil
}

func (s *careGuideService) DeleteCareInstruction(ctx context.Context, id uuid.UUID) error {
	if err := s.careInstructionRepo.DeleteCareInstruction(ctx, id); err != nil {
		if errors.Is(err, ErrNotFound) {
			return err
		}
		return fmt.Errorf("deleting care instruction: %w", err)
	}
	return nil
}

func (s *careGuideService) GetCareInstructionsForCrop(ctx context.Context, crop uuid.UUID) ([]models.CareInstruction, error) {
	instructions, err := s.careInstructionRepo.GetCareInstructionsByCrop(ctx, crop)
	if err != nil {
		return nil, fmt.Errorf("getting care instructions: %w", err)
	}
	return instructions, nil
}

func (s *careGuideService) GetCareGuideForCustomer(ctx context.Context, customer uuid.UUID) ([]models.PlotCareGuide, error) {
	rentals, err := s.rentalRepo.GetActiveRentalsByCustomer(ctx, customer)
	if err != nil {
		return nil, fmt.Errorf("getting active rentals: %w", err)
	}
	if len(rentals) == 0 {
		return []models.PlotCareGuide{}, nil
	}

	// One read for every crop being grown, not one per rental: a customer
	// renting four plots of the same crop reads that guide once.
	cropIDs := distinctCropIDs(rentals)
	instructionsByCrop, err := s.careInstructionRepo.GetCareInstructionsByCrops(ctx, cropIDs)
	if err != nil {
		return nil, fmt.Errorf("getting care instructions: %w", err)
	}

	guides := make([]models.PlotCareGuide, len(rentals))
	for i, rental := range rentals {
		guides[i] = models.PlotCareGuide{
			RentalID:     rental.ID,
			PlotID:       rental.PlotID,
			PlotName:     rental.PlotName,
			FieldName:    rental.FieldName,
			Crop:         rental.Crop,
			StartAt:      rental.StartAt,
			EndAt:        rental.EndAt,
			CurrentWeek:  rental.CurrentWeek,
			TotalWeeks:   rental.TotalWeeks,
			Instructions: instructionsWithinRental(instructionsByCrop[rental.CropID], rental.TotalWeeks),
		}
	}
	return guides, nil
}

// distinctCropIDs collects each crop exactly once, preserving the order the
// rentals came in so the query's parameter is stable across identical calls.
func distinctCropIDs(rentals []models.ActiveRental) []uuid.UUID {
	seen := make(map[uuid.UUID]struct{}, len(rentals))
	cropIDs := make([]uuid.UUID, 0, len(rentals))
	for _, rental := range rentals {
		if _, ok := seen[rental.CropID]; ok {
			continue
		}
		seen[rental.CropID] = struct{}{}
		cropIDs = append(cropIDs, rental.CropID)
	}
	return cropIDs
}

// instructionsWithinRental drops the tail of a guide the rental never reaches.
// A crop's guide is written once for the crop, but rental length comes from
// the crop's duration in *months*, so a guide that runs to week 30 against a
// 13-week rental would promise a tenant tasks for weeks that end after their
// plot is already someone else's. Always returns a non-nil slice, so the
// handler encodes `[]` rather than `null`.
func instructionsWithinRental(instructions []models.CareInstruction, totalWeeks int32) []models.CareInstruction {
	within := make([]models.CareInstruction, 0, len(instructions))
	for _, instruction := range instructions {
		if instruction.Week <= totalWeeks {
			within = append(within, instruction)
		}
	}
	return within
}
