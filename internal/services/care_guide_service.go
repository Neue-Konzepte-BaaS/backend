package services

import (
	"context"
	"errors"
	"fmt"

	"github.com/Neue-Konzepte-BaaS/backend/internal/models"
	"github.com/google/uuid"
)

// CareGuideService is the weekly care guide: per-crop instructions, read back
// by a tenant against the week their own rental is in.
//
// Every crop has one default guide, which an admin maintains and every farm
// starts from — a farmer offering tomatoes should not have to write tomato
// advice from scratch. A farmer who wants to say it differently takes the
// crop's guide over for their own farm: the first write copies the default,
// and from then on the farm's tenants read the farm's version, whatever the
// default later becomes, until the farmer resets it.
type CareGuideService interface {
	// CreateCareInstruction adds one task to a crop's guide: the default for
	// an admin, the farm's own for a farmer (taking it over first if need
	// be). Returns ErrNotFound if the crop does not exist.
	CreateCareInstruction(ctx context.Context, editor CareGuideEditor, crop uuid.UUID, week int32, title, body string) (models.CareInstruction, error)
	// UpdateCareInstruction rewrites an existing task. An admin may edit any
	// instruction. A farmer may edit their farm's own, or a default one, in
	// which case the farm takes the guide over and the farm's copy of it is
	// what changes. Returns ErrNotFound if no instruction with that id is
	// visible to the editor.
	UpdateCareInstruction(ctx context.Context, editor CareGuideEditor, id uuid.UUID, week int32, title, body string) (models.CareInstruction, error)
	// DeleteCareInstruction removes one task, under the same rules as
	// UpdateCareInstruction.
	DeleteCareInstruction(ctx context.Context, editor CareGuideEditor, id uuid.UUID) error
	// GetCareInstructionsForCrop returns one crop's whole guide, in week
	// order — the authoring view, not scoped to any rental: the default for
	// an admin, and for a farmer the version their tenants read.
	GetCareInstructionsForCrop(ctx context.Context, editor CareGuideEditor, crop uuid.UUID) (models.CropCareGuide, error)
	// ResetFarmCareGuide drops the farmer's own version of the crop's guide,
	// so their tenants read the default again. Farmers only. Returns
	// ErrNotFound if the farm has no version of its own.
	ResetFarmCareGuide(ctx context.Context, farmer, crop uuid.UUID) error
	// GetCareGuideForCustomer returns one guide per plot the customer is
	// renting right now, in the version of the farm the plot belongs to. A
	// plot whose crop has no guide yet is still returned, with an empty
	// instruction list: the tenant's own week and rental period are worth
	// showing even before anyone writes the advice.
	GetCareGuideForCustomer(ctx context.Context, customer uuid.UUID) ([]models.PlotCareGuide, error)
}

// CareGuideEditor is who is writing: an admin edits the default guide, a
// farmer their own farm's version of it.
type CareGuideEditor struct {
	AccountID uuid.UUID
	Role      models.Role
}

type careGuideService struct {
	careInstructionRepo CareInstructionRepository
	rentalRepo          RentalRepository
	farmRepo            FarmRepository
}

func NewCareGuideService(careInstructionRepo CareInstructionRepository, rentalRepo RentalRepository, farmRepo FarmRepository) CareGuideService {
	return &careGuideService{careInstructionRepo: careInstructionRepo, rentalRepo: rentalRepo, farmRepo: farmRepo}
}

// editorFarm resolves which guide an editor writes: nil for an admin (the
// default), the farmer's own farm otherwise. A farmer without a farm has no
// guide to write, which is ErrForbidden rather than the ErrNotFound the lookup
// reports — a handler would otherwise answer "crop not found".
func (s *careGuideService) editorFarm(ctx context.Context, editor CareGuideEditor) (*uuid.UUID, error) {
	if editor.Role == models.RoleAdmin {
		return nil, nil
	}
	if editor.Role != models.RoleFarmer {
		return nil, ErrForbidden
	}
	farm, err := s.farmRepo.GetFarmIDByFarmerID(ctx, editor.AccountID)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			return nil, ErrForbidden
		}
		return nil, fmt.Errorf("looking up farm: %w", err)
	}
	return &farm, nil
}

func (s *careGuideService) CreateCareInstruction(ctx context.Context, editor CareGuideEditor, crop uuid.UUID, week int32, title, body string) (models.CareInstruction, error) {
	farm, err := s.editorFarm(ctx, editor)
	if err != nil {
		return models.CareInstruction{}, err
	}
	if farm != nil {
		if err := s.careInstructionRepo.StartFarmCareGuide(ctx, crop, *farm); err != nil {
			if errors.Is(err, ErrNotFound) {
				return models.CareInstruction{}, err
			}
			return models.CareInstruction{}, fmt.Errorf("starting farm care guide: %w", err)
		}
	}

	instruction, err := s.careInstructionRepo.CreateCareInstruction(ctx, crop, farm, week, title, body)
	if err != nil {
		if errors.Is(err, ErrNotFound) || errors.Is(err, ErrInvalidCareInstruction) {
			return models.CareInstruction{}, err
		}
		return models.CareInstruction{}, fmt.Errorf("creating care instruction: %w", err)
	}
	return instruction, nil
}

// editableInstruction finds the instruction an editor's write lands on. An
// admin writes the instruction itself. A farmer writes their farm's own, or —
// for a default instruction — the farm's copy of it, taking the guide over
// first. Another farm's instruction, or a default one the farm's guide no
// longer has a copy of, is ErrNotFound: neither is anything the farmer can see.
func (s *careGuideService) editableInstruction(ctx context.Context, editor CareGuideEditor, id uuid.UUID) (uuid.UUID, error) {
	farm, err := s.editorFarm(ctx, editor)
	if err != nil {
		return uuid.Nil, err
	}
	if farm == nil {
		return id, nil
	}

	instruction, err := s.careInstructionRepo.GetCareInstructionByID(ctx, id)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			return uuid.Nil, err
		}
		return uuid.Nil, fmt.Errorf("getting care instruction: %w", err)
	}

	switch {
	case instruction.Farm != nil && *instruction.Farm == *farm:
		return id, nil
	case instruction.Farm != nil:
		return uuid.Nil, ErrNotFound
	}

	if err := s.careInstructionRepo.StartFarmCareGuide(ctx, instruction.Crop, *farm); err != nil {
		return uuid.Nil, fmt.Errorf("starting farm care guide: %w", err)
	}
	copied, err := s.careInstructionRepo.GetFarmCopyOfCareInstruction(ctx, *farm, id)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			return uuid.Nil, err
		}
		return uuid.Nil, fmt.Errorf("getting farm copy of care instruction: %w", err)
	}
	return copied.ID, nil
}

func (s *careGuideService) UpdateCareInstruction(ctx context.Context, editor CareGuideEditor, id uuid.UUID, week int32, title, body string) (models.CareInstruction, error) {
	target, err := s.editableInstruction(ctx, editor, id)
	if err != nil {
		return models.CareInstruction{}, err
	}

	instruction, err := s.careInstructionRepo.UpdateCareInstruction(ctx, target, week, title, body)
	if err != nil {
		if errors.Is(err, ErrNotFound) || errors.Is(err, ErrInvalidCareInstruction) {
			return models.CareInstruction{}, err
		}
		return models.CareInstruction{}, fmt.Errorf("updating care instruction: %w", err)
	}
	return instruction, nil
}

func (s *careGuideService) DeleteCareInstruction(ctx context.Context, editor CareGuideEditor, id uuid.UUID) error {
	target, err := s.editableInstruction(ctx, editor, id)
	if err != nil {
		return err
	}

	if err := s.careInstructionRepo.DeleteCareInstruction(ctx, target); err != nil {
		if errors.Is(err, ErrNotFound) {
			return err
		}
		return fmt.Errorf("deleting care instruction: %w", err)
	}
	return nil
}

func (s *careGuideService) GetCareInstructionsForCrop(ctx context.Context, editor CareGuideEditor, crop uuid.UUID) (models.CropCareGuide, error) {
	farm, err := s.editorFarm(ctx, editor)
	if err != nil {
		return models.CropCareGuide{}, err
	}

	if farm == nil {
		instructions, err := s.careInstructionRepo.GetDefaultCareInstructionsByCrop(ctx, crop)
		if err != nil {
			return models.CropCareGuide{}, fmt.Errorf("getting care instructions: %w", err)
		}
		return models.CropCareGuide{Instructions: instructions}, nil
	}

	farmGuide, err := s.careInstructionRepo.HasFarmCareGuide(ctx, crop, *farm)
	if err != nil {
		return models.CropCareGuide{}, fmt.Errorf("checking farm care guide: %w", err)
	}
	key := models.CropAtFarm{Crop: crop, Farm: *farm}
	byGuide, err := s.careInstructionRepo.GetEffectiveCareInstructions(ctx, []models.CropAtFarm{key})
	if err != nil {
		return models.CropCareGuide{}, fmt.Errorf("getting care instructions: %w", err)
	}
	instructions := byGuide[key]
	if instructions == nil {
		instructions = []models.CareInstruction{}
	}
	return models.CropCareGuide{FarmGuide: farmGuide, Instructions: instructions}, nil
}

func (s *careGuideService) ResetFarmCareGuide(ctx context.Context, farmer, crop uuid.UUID) error {
	farm, err := s.editorFarm(ctx, CareGuideEditor{AccountID: farmer, Role: models.RoleFarmer})
	if err != nil {
		return err
	}
	if err := s.careInstructionRepo.DeleteFarmCareGuide(ctx, crop, *farm); err != nil {
		if errors.Is(err, ErrNotFound) {
			return err
		}
		return fmt.Errorf("resetting farm care guide: %w", err)
	}
	return nil
}

func (s *careGuideService) GetCareGuideForCustomer(ctx context.Context, customer uuid.UUID) ([]models.PlotCareGuide, error) {
	rentals, err := s.rentalRepo.GetActiveRentalsByCustomer(ctx, customer)
	if err != nil {
		return nil, fmt.Errorf("getting active rentals: %w", err)
	}
	if len(rentals) == 0 {
		return []models.PlotCareGuide{}, nil
	}

	// One read for every guide being followed, not one per rental: a
	// customer renting four plots of the same crop on one farm reads that
	// guide once.
	guideKeys := distinctGuides(rentals)
	instructionsByGuide, err := s.careInstructionRepo.GetEffectiveCareInstructions(ctx, guideKeys)
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
			Instructions: instructionsWithinRental(instructionsByGuide[guideOf(rental)], rental.TotalWeeks),
		}
	}
	return guides, nil
}

// guideOf is the guide a rental's tenant reads: its crop, as grown on the
// farm the plot belongs to.
func guideOf(rental models.ActiveRental) models.CropAtFarm {
	return models.CropAtFarm{Crop: rental.CropID, Farm: rental.FarmID}
}

// distinctGuides collects each guide exactly once, preserving the order the
// rentals came in so the query's parameters are stable across identical calls.
func distinctGuides(rentals []models.ActiveRental) []models.CropAtFarm {
	seen := make(map[models.CropAtFarm]struct{}, len(rentals))
	guides := make([]models.CropAtFarm, 0, len(rentals))
	for _, rental := range rentals {
		key := guideOf(rental)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		guides = append(guides, key)
	}
	return guides
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
