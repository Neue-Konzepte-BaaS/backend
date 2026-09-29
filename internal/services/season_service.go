package services

import (
	"context"
	"errors"
	"fmt"

	"github.com/Neue-Konzepte-BaaS/backend/internal/models"
	"github.com/google/uuid"
)

// SeasonService is the recurring yearly calendar windows crops can be tied
// to: "Spring", "Summer", and so on, plus the rules tying a crop to one.
//
// There is one default set of seasons, which an admin maintains and every
// farm starts from — a farmer in a region with a longer summer should not be
// stuck with the platform's guess. A farmer may create, edit and delete their
// own seasons at any time, independent of the defaults and of any crop. Each
// crop may separately have a default season rule (an admin's choice, applying
// to every farm without an override) and any number of farm-specific rules (a
// farmer's own choice, pointing at one of that farm's own seasons) — see
// CropSeasonRule.
type SeasonService interface {
	// CreateSeason adds a season to the default set (admin) or the farmer's
	// own set (farmer).
	CreateSeason(ctx context.Context, editor SeasonEditor, name string, startMonth, startDay, endMonth, endDay int32) (models.Season, error)
	// UpdateSeason rewrites an existing season. An admin may edit any default
	// season. A farmer may edit only their own farm's seasons. Returns
	// ErrNotFound if no season with that id is visible to the editor.
	UpdateSeason(ctx context.Context, editor SeasonEditor, id uuid.UUID, name string, startMonth, startDay, endMonth, endDay int32) (models.Season, error)
	// DeleteSeason removes one season, under the same rules as UpdateSeason.
	DeleteSeason(ctx context.Context, editor SeasonEditor, id uuid.UUID) error
	// GetSeasons returns the default set (admin) or the union of the default
	// set and the farmer's own seasons (farmer) — everything they could pick
	// when assigning a crop's season.
	GetSeasons(ctx context.Context, editor SeasonEditor) ([]models.Season, error)
	// AssignCropSeason ties the crop to the season: the default rule (admin)
	// or the editor's own farm's rule (farmer), creating it if none exists yet
	// or repointing it at the given season otherwise. An admin's season must
	// be one of the defaults; a farmer's must belong to their own farm.
	// Returns ErrForbidden if the season does not belong to the editor's set,
	// ErrNotFound if the crop or season does not exist.
	AssignCropSeason(ctx context.Context, editor SeasonEditor, crop, season uuid.UUID) (models.CropSeasonRule, error)
	// RemoveCropSeasonRule drops the crop's default rule (admin) or the
	// editor's own farm's rule (farmer) for that crop, reverting it to the
	// default rule if one exists, or to unrestricted otherwise. Returns
	// ErrNotFound if no such rule exists.
	RemoveCropSeasonRule(ctx context.Context, editor SeasonEditor, crop uuid.UUID) error
	// GetCropSeasons returns every crop in the catalog paired with the season
	// it is effectively checked against for the editor: the default rule for
	// an admin, or the effective rule (their own farm's if it has one, the
	// default otherwise) for a farmer. A crop with no rule at all pairs with
	// a nil season.
	GetCropSeasons(ctx context.Context, editor SeasonEditor) ([]models.CropWithSeason, error)
}

// SeasonEditor is who is writing: an admin edits the default seasons and
// rules, a farmer their own farm's.
type SeasonEditor struct {
	AccountID uuid.UUID
	Role      models.Role
}

type seasonService struct {
	seasonRepo SeasonRepository
	farmRepo   FarmRepository
	cropRepo   CropRepository
}

func NewSeasonService(seasonRepo SeasonRepository, farmRepo FarmRepository, cropRepo CropRepository) SeasonService {
	return &seasonService{seasonRepo: seasonRepo, farmRepo: farmRepo, cropRepo: cropRepo}
}

// editorFarm resolves which set an editor writes: nil for an admin (the
// defaults), the farmer's own farm otherwise. A farmer without a farm has
// nothing to write, which is ErrForbidden rather than the ErrNotFound the
// lookup reports — a handler would otherwise answer "season not found".
func (s *seasonService) editorFarm(ctx context.Context, editor SeasonEditor) (*uuid.UUID, error) {
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

func (s *seasonService) CreateSeason(ctx context.Context, editor SeasonEditor, name string, startMonth, startDay, endMonth, endDay int32) (models.Season, error) {
	farm, err := s.editorFarm(ctx, editor)
	if err != nil {
		return models.Season{}, err
	}

	season, err := s.seasonRepo.CreateSeason(ctx, farm, name, startMonth, startDay, endMonth, endDay)
	if err != nil {
		return models.Season{}, fmt.Errorf("creating season: %w", err)
	}
	return season, nil
}

// editableSeason finds the season id an editor may write, or ErrNotFound if
// the season is not theirs to write: an admin writes any default season, a
// farmer only their own farm's.
func (s *seasonService) editableSeason(ctx context.Context, editor SeasonEditor, id uuid.UUID) (uuid.UUID, error) {
	farm, err := s.editorFarm(ctx, editor)
	if err != nil {
		return uuid.Nil, err
	}

	season, err := s.seasonRepo.GetSeasonByID(ctx, id)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			return uuid.Nil, err
		}
		return uuid.Nil, fmt.Errorf("getting season: %w", err)
	}

	if farm == nil {
		if season.Farm != nil {
			return uuid.Nil, ErrNotFound
		}
		return id, nil
	}
	if season.Farm == nil || *season.Farm != *farm {
		return uuid.Nil, ErrNotFound
	}
	return id, nil
}

func (s *seasonService) UpdateSeason(ctx context.Context, editor SeasonEditor, id uuid.UUID, name string, startMonth, startDay, endMonth, endDay int32) (models.Season, error) {
	target, err := s.editableSeason(ctx, editor, id)
	if err != nil {
		return models.Season{}, err
	}

	season, err := s.seasonRepo.UpdateSeason(ctx, target, name, startMonth, startDay, endMonth, endDay)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			return models.Season{}, err
		}
		return models.Season{}, fmt.Errorf("updating season: %w", err)
	}
	return season, nil
}

func (s *seasonService) DeleteSeason(ctx context.Context, editor SeasonEditor, id uuid.UUID) error {
	target, err := s.editableSeason(ctx, editor, id)
	if err != nil {
		return err
	}

	if err := s.seasonRepo.DeleteSeason(ctx, target); err != nil {
		if errors.Is(err, ErrNotFound) {
			return err
		}
		return fmt.Errorf("deleting season: %w", err)
	}
	return nil
}

func (s *seasonService) GetSeasons(ctx context.Context, editor SeasonEditor) ([]models.Season, error) {
	farm, err := s.editorFarm(ctx, editor)
	if err != nil {
		return nil, err
	}

	defaults, err := s.seasonRepo.GetDefaultSeasons(ctx)
	if err != nil {
		return nil, fmt.Errorf("getting default seasons: %w", err)
	}
	if farm == nil {
		return defaults, nil
	}

	own, err := s.seasonRepo.GetFarmSeasons(ctx, *farm)
	if err != nil {
		return nil, fmt.Errorf("getting farm seasons: %w", err)
	}
	return append(defaults, own...), nil
}

func (s *seasonService) AssignCropSeason(ctx context.Context, editor SeasonEditor, crop, season uuid.UUID) (models.CropSeasonRule, error) {
	farm, err := s.editorFarm(ctx, editor)
	if err != nil {
		return models.CropSeasonRule{}, err
	}

	seasonDetails, err := s.seasonRepo.GetSeasonByID(ctx, season)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			return models.CropSeasonRule{}, err
		}
		return models.CropSeasonRule{}, fmt.Errorf("getting season: %w", err)
	}
	// An admin may only tie a crop to a default season. A farmer may tie a
	// crop to either a default season or one of their own farm's — but never
	// another farm's — season; this does not let them edit or delete the
	// default season itself, only point a rule of their own at it.
	if farm == nil {
		if seasonDetails.Farm != nil {
			return models.CropSeasonRule{}, ErrForbidden
		}
	} else if seasonDetails.Farm != nil && *seasonDetails.Farm != *farm {
		return models.CropSeasonRule{}, ErrForbidden
	}

	rule, err := s.seasonRepo.CreateCropSeasonRule(ctx, crop, farm, season)
	if err != nil {
		if errors.Is(err, ErrCropSeasonRuleExists) {
			existing, getErr := s.seasonRepo.GetCropSeasonRuleForCrop(ctx, crop, farm)
			if getErr != nil {
				return models.CropSeasonRule{}, fmt.Errorf("getting existing crop season rule: %w", getErr)
			}
			rule, err = s.seasonRepo.UpdateCropSeasonRule(ctx, existing.ID, season)
			if err != nil {
				return models.CropSeasonRule{}, fmt.Errorf("updating crop season rule: %w", err)
			}
			return rule, nil
		}
		if errors.Is(err, ErrNotFound) {
			return models.CropSeasonRule{}, err
		}
		return models.CropSeasonRule{}, fmt.Errorf("creating crop season rule: %w", err)
	}
	return rule, nil
}

func (s *seasonService) RemoveCropSeasonRule(ctx context.Context, editor SeasonEditor, crop uuid.UUID) error {
	farm, err := s.editorFarm(ctx, editor)
	if err != nil {
		return err
	}

	rule, err := s.seasonRepo.GetCropSeasonRuleForCrop(ctx, crop, farm)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			return err
		}
		return fmt.Errorf("getting crop season rule: %w", err)
	}

	if err := s.seasonRepo.DeleteCropSeasonRule(ctx, rule.ID); err != nil {
		if errors.Is(err, ErrNotFound) {
			return err
		}
		return fmt.Errorf("deleting crop season rule: %w", err)
	}
	return nil
}

// noFarmSentinel stands in for "no farm" in GetEffectiveSeasonsForCrops when
// resolving an admin's view: real farm ids are never uuid.Nil, so the
// override branch of that query can never match it, leaving only each crop's
// default rule (exactly what an admin's view should show).
var noFarmSentinel = uuid.UUID{}

func (s *seasonService) GetCropSeasons(ctx context.Context, editor SeasonEditor) ([]models.CropWithSeason, error) {
	farm, err := s.editorFarm(ctx, editor)
	if err != nil {
		return nil, err
	}

	crops, err := s.cropRepo.GetAllCrops(ctx)
	if err != nil {
		return nil, fmt.Errorf("getting crops: %w", err)
	}

	resolveFarm := noFarmSentinel
	if farm != nil {
		resolveFarm = *farm
	}
	pairs := make([]models.CropAtFarm, len(crops))
	for i, crop := range crops {
		pairs[i] = models.CropAtFarm{Crop: crop.ID, Farm: resolveFarm}
	}
	seasons, err := s.seasonRepo.GetEffectiveSeasonsForCrops(ctx, pairs)
	if err != nil {
		return nil, fmt.Errorf("getting effective seasons: %w", err)
	}

	result := make([]models.CropWithSeason, len(crops))
	for i, crop := range crops {
		result[i] = models.CropWithSeason{Crop: crop}
		if season, ok := seasons[models.CropAtFarm{Crop: crop.ID, Farm: resolveFarm}]; ok {
			s := season
			result[i].Season = &s
		}
	}
	return result, nil
}
