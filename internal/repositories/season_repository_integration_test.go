package repositories_test

import (
	"context"
	"testing"

	"github.com/Neue-Konzepte-BaaS/backend/internal/repositories"
	database "github.com/Neue-Konzepte-BaaS/backend/internal/repositories/db"
	"github.com/google/uuid"
)

// TestSeasonDefaultsAreSeeded checks the four seasons the seasons migration
// seeds are present and readable as the global default set.
func TestSeasonDefaultsAreSeeded(t *testing.T) {
	pool := setupTestDB(t)
	ctx := context.Background()

	seasonRepo := repositories.NewSeasonRepository(pool, database.New(pool))

	seasons, err := seasonRepo.GetDefaultSeasons(ctx)
	if err != nil {
		t.Fatalf("getting default seasons: %v", err)
	}
	if len(seasons) != 4 {
		t.Fatalf("got %d default seasons, want 4", len(seasons))
	}
	for _, season := range seasons {
		if season.Farm != nil {
			t.Errorf("default season %q has farm %v, want nil", season.Name, *season.Farm)
		}
	}
}

// TestGetEffectiveSeasonForCrop covers the three resolution cases: a farm's
// own rule wins over the default, the default is used when the farm has no
// rule of its own, and a crop with no rule at all is unrestricted.
func TestGetEffectiveSeasonForCrop(t *testing.T) {
	pool := setupTestDB(t)
	ctx := context.Background()

	queries := database.New(pool)
	seasonRepo := repositories.NewSeasonRepository(pool, queries)
	_, farm, _, crop := seedFarmerWithPlots(t, ctx, pool, 1)

	// No rule at all: unrestricted.
	_, ok, err := seasonRepo.GetEffectiveSeasonForCrop(ctx, crop, farm)
	if err != nil {
		t.Fatalf("getting effective season with no rule: %v", err)
	}
	if ok {
		t.Error("crop with no rule reports restricted, want unrestricted")
	}

	defaultSeason, err := seasonRepo.CreateSeason(ctx, nil, "Spring-"+uuid.NewString(), 3, 1, 5, 31)
	if err != nil {
		t.Fatalf("creating default season: %v", err)
	}
	if _, err := seasonRepo.CreateCropSeasonRule(ctx, crop, nil, defaultSeason.ID); err != nil {
		t.Fatalf("creating default crop season rule: %v", err)
	}

	// The default rule applies before the farm has one of its own.
	effective, ok, err := seasonRepo.GetEffectiveSeasonForCrop(ctx, crop, farm)
	if err != nil || !ok || effective.ID != defaultSeason.ID {
		t.Fatalf("effective season = %+v, ok:%v, %v; want the default %v", effective, ok, err, defaultSeason.ID)
	}

	ownSeason, err := seasonRepo.CreateSeason(ctx, &farm, "Monsoon-"+uuid.NewString(), 7, 1, 9, 30)
	if err != nil {
		t.Fatalf("creating farm season: %v", err)
	}
	if _, err := seasonRepo.CreateCropSeasonRule(ctx, crop, &farm, ownSeason.ID); err != nil {
		t.Fatalf("creating farm crop season rule: %v", err)
	}

	// The farm's own rule now wins.
	effective, ok, err = seasonRepo.GetEffectiveSeasonForCrop(ctx, crop, farm)
	if err != nil || !ok || effective.ID != ownSeason.ID {
		t.Fatalf("effective season = %+v, ok:%v, %v; want the farm's own %v", effective, ok, err, ownSeason.ID)
	}

	// A different farm still reads the default.
	otherFarm := uuid.New()
	effective, ok, err = seasonRepo.GetEffectiveSeasonForCrop(ctx, crop, otherFarm)
	if err != nil || !ok || effective.ID != defaultSeason.ID {
		t.Fatalf("other farm's effective season = %+v, ok:%v, %v; want the default %v", effective, ok, err, defaultSeason.ID)
	}
}

// TestCropSeasonRuleUniquePerFarm checks the idx_season_crop_crop_farm index
// rejects a second rule for the same crop in the same set (the defaults, or
// one farm's own), while allowing one rule per distinct set.
func TestCropSeasonRuleUniquePerFarm(t *testing.T) {
	pool := setupTestDB(t)
	ctx := context.Background()

	queries := database.New(pool)
	seasonRepo := repositories.NewSeasonRepository(pool, queries)
	_, farm, _, crop := seedFarmerWithPlots(t, ctx, pool, 1)

	seasonA, err := seasonRepo.CreateSeason(ctx, nil, "Spring-"+uuid.NewString(), 3, 1, 5, 31)
	if err != nil {
		t.Fatalf("creating season: %v", err)
	}
	seasonB, err := seasonRepo.CreateSeason(ctx, nil, "Summer-"+uuid.NewString(), 6, 1, 8, 31)
	if err != nil {
		t.Fatalf("creating season: %v", err)
	}

	if _, err := seasonRepo.CreateCropSeasonRule(ctx, crop, nil, seasonA.ID); err != nil {
		t.Fatalf("creating first default rule: %v", err)
	}
	if _, err := seasonRepo.CreateCropSeasonRule(ctx, crop, nil, seasonB.ID); err == nil {
		t.Error("creating a second default rule for the same crop succeeded, want a unique violation")
	}

	// A rule for the same crop under a different set (this farm's own) is
	// allowed.
	ownSeason, err := seasonRepo.CreateSeason(ctx, &farm, "Own-"+uuid.NewString(), 1, 1, 2, 1)
	if err != nil {
		t.Fatalf("creating farm season: %v", err)
	}
	if _, err := seasonRepo.CreateCropSeasonRule(ctx, crop, &farm, ownSeason.ID); err != nil {
		t.Errorf("creating a rule for the same crop in a different set: %v", err)
	}
}
