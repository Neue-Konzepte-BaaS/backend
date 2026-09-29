package services

import (
	"context"
	"errors"
	"testing"

	"github.com/Neue-Konzepte-BaaS/backend/internal/models"
	"github.com/google/uuid"
)

// fakeSeasonServiceRepo is an in-memory season/season_crop table: enough of
// the real semantics — direct CRUD scoped by farm ownership, and crop-season
// rules resolved by exact set or by fallback — for the service's rules to be
// tested against something that behaves like the database.
//
// Named distinctly from the rental service's fakeSeasonRepo (same package,
// narrower fake) since this one needs to model writes too.
type fakeSeasonServiceRepo struct {
	seasons []models.Season
	rules   []models.CropSeasonRule
	err     error
}

func newFakeSeasonRepo(seasons ...models.Season) *fakeSeasonServiceRepo {
	return &fakeSeasonServiceRepo{seasons: seasons}
}

func (f *fakeSeasonServiceRepo) find(id uuid.UUID) (int, bool) {
	for i, season := range f.seasons {
		if season.ID == id {
			return i, true
		}
	}
	return 0, false
}

func (f *fakeSeasonServiceRepo) CreateSeason(_ context.Context, farm *uuid.UUID, name string, startMonth, startDay, endMonth, endDay int32) (models.Season, error) {
	if f.err != nil {
		return models.Season{}, f.err
	}
	season := models.Season{ID: uuid.New(), Farm: farm, Name: name, StartMonth: startMonth, StartDay: startDay, EndMonth: endMonth, EndDay: endDay}
	f.seasons = append(f.seasons, season)
	return season, nil
}

func (f *fakeSeasonServiceRepo) UpdateSeason(_ context.Context, id uuid.UUID, name string, startMonth, startDay, endMonth, endDay int32) (models.Season, error) {
	if f.err != nil {
		return models.Season{}, f.err
	}
	i, ok := f.find(id)
	if !ok {
		return models.Season{}, ErrNotFound
	}
	f.seasons[i].Name, f.seasons[i].StartMonth, f.seasons[i].StartDay, f.seasons[i].EndMonth, f.seasons[i].EndDay = name, startMonth, startDay, endMonth, endDay
	return f.seasons[i], nil
}

func (f *fakeSeasonServiceRepo) DeleteSeason(_ context.Context, id uuid.UUID) error {
	if f.err != nil {
		return f.err
	}
	i, ok := f.find(id)
	if !ok {
		return ErrNotFound
	}
	f.seasons = append(f.seasons[:i], f.seasons[i+1:]...)
	return nil
}

func (f *fakeSeasonServiceRepo) GetSeasonByID(_ context.Context, id uuid.UUID) (models.Season, error) {
	if f.err != nil {
		return models.Season{}, f.err
	}
	i, ok := f.find(id)
	if !ok {
		return models.Season{}, ErrNotFound
	}
	return f.seasons[i], nil
}

func (f *fakeSeasonServiceRepo) GetDefaultSeasons(_ context.Context) ([]models.Season, error) {
	if f.err != nil {
		return nil, f.err
	}
	var out []models.Season
	for _, season := range f.seasons {
		if season.Farm == nil {
			out = append(out, season)
		}
	}
	return out, nil
}

func (f *fakeSeasonServiceRepo) GetFarmSeasons(_ context.Context, farm uuid.UUID) ([]models.Season, error) {
	if f.err != nil {
		return nil, f.err
	}
	var out []models.Season
	for _, season := range f.seasons {
		if season.Farm != nil && *season.Farm == farm {
			out = append(out, season)
		}
	}
	return out, nil
}

func sameFarm(a, b *uuid.UUID) bool {
	if a == nil || b == nil {
		return a == b
	}
	return *a == *b
}

func (f *fakeSeasonServiceRepo) findRule(crop uuid.UUID, farm *uuid.UUID) (int, bool) {
	for i, rule := range f.rules {
		if rule.Crop == crop && sameFarm(rule.Farm, farm) {
			return i, true
		}
	}
	return 0, false
}

func (f *fakeSeasonServiceRepo) CreateCropSeasonRule(_ context.Context, crop uuid.UUID, farm *uuid.UUID, season uuid.UUID) (models.CropSeasonRule, error) {
	if f.err != nil {
		return models.CropSeasonRule{}, f.err
	}
	if _, ok := f.findRule(crop, farm); ok {
		return models.CropSeasonRule{}, ErrCropSeasonRuleExists
	}
	rule := models.CropSeasonRule{ID: uuid.New(), Crop: crop, Season: season, Farm: farm}
	f.rules = append(f.rules, rule)
	return rule, nil
}

func (f *fakeSeasonServiceRepo) UpdateCropSeasonRule(_ context.Context, id uuid.UUID, season uuid.UUID) (models.CropSeasonRule, error) {
	if f.err != nil {
		return models.CropSeasonRule{}, f.err
	}
	for i, rule := range f.rules {
		if rule.ID == id {
			f.rules[i].Season = season
			return f.rules[i], nil
		}
	}
	return models.CropSeasonRule{}, ErrNotFound
}

func (f *fakeSeasonServiceRepo) DeleteCropSeasonRule(_ context.Context, id uuid.UUID) error {
	if f.err != nil {
		return f.err
	}
	for i, rule := range f.rules {
		if rule.ID == id {
			f.rules = append(f.rules[:i], f.rules[i+1:]...)
			return nil
		}
	}
	return ErrNotFound
}

func (f *fakeSeasonServiceRepo) GetCropSeasonRuleByID(_ context.Context, id uuid.UUID) (models.CropSeasonRule, error) {
	if f.err != nil {
		return models.CropSeasonRule{}, f.err
	}
	for _, rule := range f.rules {
		if rule.ID == id {
			return rule, nil
		}
	}
	return models.CropSeasonRule{}, ErrNotFound
}

func (f *fakeSeasonServiceRepo) GetCropSeasonRuleForCrop(_ context.Context, crop uuid.UUID, farm *uuid.UUID) (models.CropSeasonRule, error) {
	if f.err != nil {
		return models.CropSeasonRule{}, f.err
	}
	i, ok := f.findRule(crop, farm)
	if !ok {
		return models.CropSeasonRule{}, ErrNotFound
	}
	return f.rules[i], nil
}

func (f *fakeSeasonServiceRepo) GetEffectiveSeasonForCrop(_ context.Context, crop, farm uuid.UUID) (models.Season, bool, error) {
	if f.err != nil {
		return models.Season{}, false, f.err
	}
	farmCopy := farm
	if i, ok := f.findRule(crop, &farmCopy); ok {
		season, sok := f.find(f.rules[i].Season)
		if sok {
			return f.seasons[season], true, nil
		}
	}
	if i, ok := f.findRule(crop, nil); ok {
		season, sok := f.find(f.rules[i].Season)
		if sok {
			return f.seasons[season], true, nil
		}
	}
	return models.Season{}, false, nil
}

// GetEffectiveSeasonsForCrops mirrors GetEffectiveSeasonForCrop, batched: a
// pair with no rule at all is simply absent from the result, same as the
// real repository.
func (f *fakeSeasonServiceRepo) GetEffectiveSeasonsForCrops(ctx context.Context, pairs []models.CropAtFarm) (map[models.CropAtFarm]models.Season, error) {
	if f.err != nil {
		return nil, f.err
	}
	out := make(map[models.CropAtFarm]models.Season, len(pairs))
	for _, pair := range pairs {
		if season, ok, _ := f.GetEffectiveSeasonForCrop(ctx, pair.Crop, pair.Farm); ok {
			out[pair] = season
		}
	}
	return out, nil
}

// fakeSeasonServiceCropRepo is an in-memory CropRepository, only
// implementing what seasonService.GetCropSeasons needs.
type fakeSeasonServiceCropRepo struct {
	crops []models.Crop
}

func (f *fakeSeasonServiceCropRepo) CreateCrop(context.Context, string, int32) (models.Crop, error) {
	return models.Crop{}, nil
}
func (f *fakeSeasonServiceCropRepo) UpdateCrop(context.Context, uuid.UUID, string, int32) (models.Crop, error) {
	return models.Crop{}, nil
}
func (f *fakeSeasonServiceCropRepo) DeleteCrop(context.Context, uuid.UUID) error { return nil }
func (f *fakeSeasonServiceCropRepo) GetAllCrops(context.Context) ([]models.Crop, error) {
	return f.crops, nil
}
func (f *fakeSeasonServiceCropRepo) GetCropByID(context.Context, uuid.UUID) (models.Crop, error) {
	return models.Crop{}, nil
}
func (f *fakeSeasonServiceCropRepo) SetPlotCrops(context.Context, uuid.UUID, int32, []uuid.UUID) error {
	return nil
}
func (f *fakeSeasonServiceCropRepo) GetCropsByPlot(context.Context, uuid.UUID) ([]models.Crop, error) {
	return nil, nil
}
func (f *fakeSeasonServiceCropRepo) GetCropsByPlots(context.Context, []uuid.UUID) (map[uuid.UUID][]models.Crop, error) {
	return nil, nil
}
func (f *fakeSeasonServiceCropRepo) GetPricedCropOfferingsByPlots(context.Context, []uuid.UUID) (map[uuid.UUID][]models.PlotCropOffering, error) {
	return nil, nil
}

// seasonFixture is one farmer with a farm, and an admin.
type seasonFixture struct {
	farmer, farm uuid.UUID
	farmRepo     *fakeFarmRepo
	cropRepo     *fakeSeasonServiceCropRepo
}

func newSeasonFixture() seasonFixture {
	farmer, farm := uuid.New(), uuid.New()
	return seasonFixture{
		farmer:   farmer,
		farm:     farm,
		farmRepo: &fakeFarmRepo{farmIDByFarmer: map[uuid.UUID]uuid.UUID{farmer: farm}},
		cropRepo: &fakeSeasonServiceCropRepo{},
	}
}

func (c seasonFixture) farmerEditor() SeasonEditor {
	return SeasonEditor{AccountID: c.farmer, Role: models.RoleFarmer}
}

var seasonAdminEditor = SeasonEditor{AccountID: uuid.New(), Role: models.RoleAdmin}

func seasonNames(seasons []models.Season) []string {
	out := make([]string, len(seasons))
	for i, season := range seasons {
		out[i] = season.Name
	}
	return out
}

func TestSeasonAuthoring(t *testing.T) {
	ctx := context.Background()

	t.Run("an admin writes the default set", func(t *testing.T) {
		c := newSeasonFixture()
		repo := newFakeSeasonRepo()
		service := NewSeasonService(repo, c.farmRepo, c.cropRepo)

		created, err := service.CreateSeason(ctx, seasonAdminEditor, "Spring", 3, 1, 5, 31)
		if err != nil {
			t.Fatalf("CreateSeason: %v", err)
		}
		if created.Farm != nil {
			t.Errorf("admin's season has farm %v, want the default set", *created.Farm)
		}
	})

	t.Run("a farmer creates their own season independent of the defaults", func(t *testing.T) {
		c := newSeasonFixture()
		repo := newFakeSeasonRepo(
			models.Season{ID: uuid.New(), Name: "Spring", StartMonth: 3, StartDay: 1, EndMonth: 5, EndDay: 31},
		)
		service := NewSeasonService(repo, c.farmRepo, c.cropRepo)

		created, err := service.CreateSeason(ctx, c.farmerEditor(), "Monsoon", 7, 1, 9, 30)
		if err != nil {
			t.Fatalf("CreateSeason: %v", err)
		}
		if created.Farm == nil || *created.Farm != c.farm {
			t.Fatalf("farmer's season has farm %v, want %v", created.Farm, c.farm)
		}

		farmView, err := service.GetSeasons(ctx, c.farmerEditor())
		if err != nil {
			t.Fatalf("GetSeasons: %v", err)
		}
		if got, want := seasonNames(farmView), []string{"Spring", "Monsoon"}; !equalStrings(got, want) {
			t.Errorf("farmer's view = %v, want defaults plus their own %v", got, want)
		}

		defaultSeasons, err := service.GetSeasons(ctx, seasonAdminEditor)
		if err != nil {
			t.Fatalf("GetSeasons: %v", err)
		}
		if got, want := seasonNames(defaultSeasons), []string{"Spring"}; !equalStrings(got, want) {
			t.Errorf("default seasons = %v, want them untouched %v", got, want)
		}
	})

	t.Run("a farmer edits only their own season", func(t *testing.T) {
		c := newSeasonFixture()
		own := models.Season{ID: uuid.New(), Farm: &c.farm, Name: "Spring", StartMonth: 3, StartDay: 1, EndMonth: 5, EndDay: 31}
		repo := newFakeSeasonRepo(own)
		service := NewSeasonService(repo, c.farmRepo, c.cropRepo)

		updated, err := service.UpdateSeason(ctx, c.farmerEditor(), own.ID, "Spring", 2, 15, 5, 31)
		if err != nil {
			t.Fatalf("UpdateSeason: %v", err)
		}
		if updated.StartMonth != 2 || updated.StartDay != 15 {
			t.Errorf("updated season = %+v, want the new dates", updated)
		}
	})

	t.Run("a farmer cannot edit a default season", func(t *testing.T) {
		c := newSeasonFixture()
		defaultSeason := models.Season{ID: uuid.New(), Name: "Spring", StartMonth: 3, StartDay: 1, EndMonth: 5, EndDay: 31}
		repo := newFakeSeasonRepo(defaultSeason)
		service := NewSeasonService(repo, c.farmRepo, c.cropRepo)

		if _, err := service.UpdateSeason(ctx, c.farmerEditor(), defaultSeason.ID, "Spring", 2, 1, 5, 31); !errors.Is(err, ErrNotFound) {
			t.Errorf("err = %v, want ErrNotFound", err)
		}
		if err := service.DeleteSeason(ctx, c.farmerEditor(), defaultSeason.ID); !errors.Is(err, ErrNotFound) {
			t.Errorf("err = %v, want ErrNotFound", err)
		}
	})

	t.Run("a farmer cannot touch another farm's season", func(t *testing.T) {
		c := newSeasonFixture()
		otherFarm := uuid.New()
		theirs := models.Season{ID: uuid.New(), Farm: &otherFarm, Name: "Theirs", StartMonth: 1, StartDay: 1, EndMonth: 2, EndDay: 1}
		repo := newFakeSeasonRepo(theirs)
		service := NewSeasonService(repo, c.farmRepo, c.cropRepo)

		if _, err := service.UpdateSeason(ctx, c.farmerEditor(), theirs.ID, "Mine now", 1, 1, 2, 1); !errors.Is(err, ErrNotFound) {
			t.Errorf("update err = %v, want ErrNotFound", err)
		}
		if err := service.DeleteSeason(ctx, c.farmerEditor(), theirs.ID); !errors.Is(err, ErrNotFound) {
			t.Errorf("delete err = %v, want ErrNotFound", err)
		}
	})

	t.Run("a farmer deleting their own season leaves the defaults alone", func(t *testing.T) {
		c := newSeasonFixture()
		defaultSeason := models.Season{ID: uuid.New(), Name: "Spring", StartMonth: 3, StartDay: 1, EndMonth: 5, EndDay: 31}
		own := models.Season{ID: uuid.New(), Farm: &c.farm, Name: "Monsoon", StartMonth: 7, StartDay: 1, EndMonth: 9, EndDay: 30}
		repo := newFakeSeasonRepo(defaultSeason, own)
		service := NewSeasonService(repo, c.farmRepo, c.cropRepo)

		if err := service.DeleteSeason(ctx, c.farmerEditor(), own.ID); err != nil {
			t.Fatalf("DeleteSeason: %v", err)
		}

		farmView, err := service.GetSeasons(ctx, c.farmerEditor())
		if err != nil {
			t.Fatalf("GetSeasons: %v", err)
		}
		if got, want := seasonNames(farmView), []string{"Spring"}; !equalStrings(got, want) {
			t.Errorf("farmer's view after delete = %v, want just the default %v", got, want)
		}
	})

	t.Run("a farmer without a farm is forbidden", func(t *testing.T) {
		service := NewSeasonService(newFakeSeasonRepo(), &fakeFarmRepo{}, &fakeSeasonServiceCropRepo{})
		farmless := SeasonEditor{AccountID: uuid.New(), Role: models.RoleFarmer}

		if _, err := service.CreateSeason(ctx, farmless, "Spring", 3, 1, 5, 31); !errors.Is(err, ErrForbidden) {
			t.Errorf("err = %v, want ErrForbidden", err)
		}
	})

	t.Run("a customer is never an editor", func(t *testing.T) {
		service := NewSeasonService(newFakeSeasonRepo(), &fakeFarmRepo{}, &fakeSeasonServiceCropRepo{})
		customer := SeasonEditor{AccountID: uuid.New(), Role: models.RoleCustomer}

		if _, err := service.GetSeasons(ctx, customer); !errors.Is(err, ErrForbidden) {
			t.Errorf("err = %v, want ErrForbidden", err)
		}
	})
}

func TestCropSeasonAssignment(t *testing.T) {
	ctx := context.Background()
	crop := uuid.New()

	t.Run("an admin sets the default rule", func(t *testing.T) {
		c := newSeasonFixture()
		spring := models.Season{ID: uuid.New(), Name: "Spring", StartMonth: 3, StartDay: 1, EndMonth: 5, EndDay: 31}
		repo := newFakeSeasonRepo(spring)
		service := NewSeasonService(repo, c.farmRepo, c.cropRepo)

		rule, err := service.AssignCropSeason(ctx, seasonAdminEditor, crop, spring.ID)
		if err != nil {
			t.Fatalf("AssignCropSeason: %v", err)
		}
		if rule.Farm != nil {
			t.Errorf("admin's rule has farm %v, want the default rule", *rule.Farm)
		}
		if rule.Season != spring.ID {
			t.Errorf("rule season = %v, want %v", rule.Season, spring.ID)
		}
	})

	t.Run("re-assigning repoints the existing rule instead of erroring", func(t *testing.T) {
		c := newSeasonFixture()
		spring := models.Season{ID: uuid.New(), Name: "Spring", StartMonth: 3, StartDay: 1, EndMonth: 5, EndDay: 31}
		summer := models.Season{ID: uuid.New(), Name: "Summer", StartMonth: 6, StartDay: 1, EndMonth: 8, EndDay: 31}
		repo := newFakeSeasonRepo(spring, summer)
		service := NewSeasonService(repo, c.farmRepo, c.cropRepo)

		first, err := service.AssignCropSeason(ctx, seasonAdminEditor, crop, spring.ID)
		if err != nil {
			t.Fatalf("AssignCropSeason: %v", err)
		}
		second, err := service.AssignCropSeason(ctx, seasonAdminEditor, crop, summer.ID)
		if err != nil {
			t.Fatalf("second AssignCropSeason: %v", err)
		}
		if second.ID != first.ID {
			t.Errorf("second assignment created a new rule %v, want it to repoint %v", second.ID, first.ID)
		}
		if second.Season != summer.ID {
			t.Errorf("rule season = %v, want %v", second.Season, summer.ID)
		}
	})

	t.Run("a farmer assigns their own farm's rule to their own season", func(t *testing.T) {
		c := newSeasonFixture()
		own := models.Season{ID: uuid.New(), Farm: &c.farm, Name: "Spring", StartMonth: 3, StartDay: 1, EndMonth: 5, EndDay: 31}
		repo := newFakeSeasonRepo(own)
		service := NewSeasonService(repo, c.farmRepo, c.cropRepo)

		rule, err := service.AssignCropSeason(ctx, c.farmerEditor(), crop, own.ID)
		if err != nil {
			t.Fatalf("AssignCropSeason: %v", err)
		}
		if rule.Farm == nil || *rule.Farm != c.farm {
			t.Errorf("rule farm = %v, want %v", rule.Farm, c.farm)
		}
	})

	t.Run("a farmer assigns a default season to their crop", func(t *testing.T) {
		c := newSeasonFixture()
		defaultSeason := models.Season{ID: uuid.New(), Name: "Spring", StartMonth: 3, StartDay: 1, EndMonth: 5, EndDay: 31}
		repo := newFakeSeasonRepo(defaultSeason)
		service := NewSeasonService(repo, c.farmRepo, c.cropRepo)

		rule, err := service.AssignCropSeason(ctx, c.farmerEditor(), crop, defaultSeason.ID)
		if err != nil {
			t.Fatalf("AssignCropSeason: %v", err)
		}
		if rule.Farm == nil || *rule.Farm != c.farm {
			t.Errorf("rule farm = %v, want the farmer's own farm %v (the rule, not the season, scopes to the farmer)", rule.Farm, c.farm)
		}
		if rule.Season != defaultSeason.ID {
			t.Errorf("rule season = %v, want %v", rule.Season, defaultSeason.ID)
		}
	})

	t.Run("a farmer cannot assign a rule pointing at another farm's season", func(t *testing.T) {
		c := newSeasonFixture()
		otherFarm := uuid.New()
		theirs := models.Season{ID: uuid.New(), Farm: &otherFarm, Name: "Theirs", StartMonth: 1, StartDay: 1, EndMonth: 2, EndDay: 1}
		repo := newFakeSeasonRepo(theirs)
		service := NewSeasonService(repo, c.farmRepo, c.cropRepo)

		if _, err := service.AssignCropSeason(ctx, c.farmerEditor(), crop, theirs.ID); !errors.Is(err, ErrForbidden) {
			t.Errorf("err = %v, want ErrForbidden", err)
		}
	})

	t.Run("an admin cannot assign a rule pointing at a farm's season", func(t *testing.T) {
		c := newSeasonFixture()
		own := models.Season{ID: uuid.New(), Farm: &c.farm, Name: "Spring", StartMonth: 3, StartDay: 1, EndMonth: 5, EndDay: 31}
		repo := newFakeSeasonRepo(own)
		service := NewSeasonService(repo, c.farmRepo, c.cropRepo)

		if _, err := service.AssignCropSeason(ctx, seasonAdminEditor, crop, own.ID); !errors.Is(err, ErrForbidden) {
			t.Errorf("err = %v, want ErrForbidden", err)
		}
	})

	t.Run("removing a farm's rule falls back to the default for that farm", func(t *testing.T) {
		c := newSeasonFixture()
		defaultSeason := models.Season{ID: uuid.New(), Name: "Spring", StartMonth: 3, StartDay: 1, EndMonth: 5, EndDay: 31}
		own := models.Season{ID: uuid.New(), Farm: &c.farm, Name: "Monsoon", StartMonth: 7, StartDay: 1, EndMonth: 9, EndDay: 30}
		repo := newFakeSeasonRepo(defaultSeason, own)
		service := NewSeasonService(repo, c.farmRepo, c.cropRepo)

		if _, err := service.AssignCropSeason(ctx, seasonAdminEditor, crop, defaultSeason.ID); err != nil {
			t.Fatalf("admin AssignCropSeason: %v", err)
		}
		if _, err := service.AssignCropSeason(ctx, c.farmerEditor(), crop, own.ID); err != nil {
			t.Fatalf("farmer AssignCropSeason: %v", err)
		}

		effective, ok, err := repo.GetEffectiveSeasonForCrop(ctx, crop, c.farm)
		if err != nil || !ok || effective.ID != own.ID {
			t.Fatalf("effective season before removal = %+v, %v, %v; want the farm's own %v", effective, ok, err, own.ID)
		}

		if err := service.RemoveCropSeasonRule(ctx, c.farmerEditor(), crop); err != nil {
			t.Fatalf("RemoveCropSeasonRule: %v", err)
		}

		effective, ok, err = repo.GetEffectiveSeasonForCrop(ctx, crop, c.farm)
		if err != nil || !ok || effective.ID != defaultSeason.ID {
			t.Fatalf("effective season after removal = %+v, %v, %v; want the default %v", effective, ok, err, defaultSeason.ID)
		}
	})

	t.Run("removing the default rule when nothing else exists leaves the crop unrestricted", func(t *testing.T) {
		c := newSeasonFixture()
		defaultSeason := models.Season{ID: uuid.New(), Name: "Spring", StartMonth: 3, StartDay: 1, EndMonth: 5, EndDay: 31}
		repo := newFakeSeasonRepo(defaultSeason)
		service := NewSeasonService(repo, c.farmRepo, c.cropRepo)

		if _, err := service.AssignCropSeason(ctx, seasonAdminEditor, crop, defaultSeason.ID); err != nil {
			t.Fatalf("AssignCropSeason: %v", err)
		}
		if err := service.RemoveCropSeasonRule(ctx, seasonAdminEditor, crop); err != nil {
			t.Fatalf("RemoveCropSeasonRule: %v", err)
		}

		_, ok, err := repo.GetEffectiveSeasonForCrop(ctx, crop, c.farm)
		if err != nil || ok {
			t.Fatalf("effective season = ok:%v, %v; want unrestricted", ok, err)
		}
	})

	t.Run("removing a rule that does not exist is not found", func(t *testing.T) {
		c := newSeasonFixture()
		repo := newFakeSeasonRepo()
		service := NewSeasonService(repo, c.farmRepo, c.cropRepo)

		if err := service.RemoveCropSeasonRule(ctx, c.farmerEditor(), crop); !errors.Is(err, ErrNotFound) {
			t.Errorf("err = %v, want ErrNotFound", err)
		}
	})
}

func TestGetCropSeasons(t *testing.T) {
	ctx := context.Background()

	t.Run("an admin sees each crop's default rule, unrestricted crops paired with nil", func(t *testing.T) {
		c := newSeasonFixture()
		spring := models.Season{ID: uuid.New(), Name: "Spring", StartMonth: 3, StartDay: 1, EndMonth: 5, EndDay: 31}
		restricted, unrestricted := uuid.New(), uuid.New()
		repo := newFakeSeasonRepo(spring)
		c.cropRepo.crops = []models.Crop{{ID: restricted, Name: "Karotte"}, {ID: unrestricted, Name: "Tomate"}}
		service := NewSeasonService(repo, c.farmRepo, c.cropRepo)

		if _, err := service.AssignCropSeason(ctx, seasonAdminEditor, restricted, spring.ID); err != nil {
			t.Fatalf("AssignCropSeason: %v", err)
		}

		result, err := service.GetCropSeasons(ctx, seasonAdminEditor)
		if err != nil {
			t.Fatalf("GetCropSeasons: %v", err)
		}
		if len(result) != 2 {
			t.Fatalf("len(result) = %d, want 2", len(result))
		}
		byID := make(map[uuid.UUID]models.CropWithSeason, len(result))
		for _, r := range result {
			byID[r.Crop.ID] = r
		}
		if byID[restricted].Season == nil || byID[restricted].Season.ID != spring.ID {
			t.Errorf("restricted crop's season = %+v, want %v", byID[restricted].Season, spring.ID)
		}
		if byID[unrestricted].Season != nil {
			t.Errorf("unrestricted crop's season = %+v, want nil", byID[unrestricted].Season)
		}
	})

	t.Run("a farmer sees the effective season: their own rule over the default", func(t *testing.T) {
		c := newSeasonFixture()
		crop := uuid.New()
		defaultSeason := models.Season{ID: uuid.New(), Name: "Spring", StartMonth: 3, StartDay: 1, EndMonth: 5, EndDay: 31}
		own := models.Season{ID: uuid.New(), Farm: &c.farm, Name: "Monsoon", StartMonth: 7, StartDay: 1, EndMonth: 9, EndDay: 30}
		repo := newFakeSeasonRepo(defaultSeason, own)
		c.cropRepo.crops = []models.Crop{{ID: crop, Name: "Karotte"}}
		service := NewSeasonService(repo, c.farmRepo, c.cropRepo)

		if _, err := service.AssignCropSeason(ctx, seasonAdminEditor, crop, defaultSeason.ID); err != nil {
			t.Fatalf("admin AssignCropSeason: %v", err)
		}
		if _, err := service.AssignCropSeason(ctx, c.farmerEditor(), crop, own.ID); err != nil {
			t.Fatalf("farmer AssignCropSeason: %v", err)
		}

		result, err := service.GetCropSeasons(ctx, c.farmerEditor())
		if err != nil {
			t.Fatalf("GetCropSeasons: %v", err)
		}
		if len(result) != 1 || result[0].Season == nil || result[0].Season.ID != own.ID {
			t.Fatalf("result = %+v, want one crop with season %v", result, own.ID)
		}
	})
}
