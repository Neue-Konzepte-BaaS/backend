package repositories_test

import (
	"context"
	"errors"
	"testing"

	"github.com/Neue-Konzepte-BaaS/backend/internal/repositories"
	database "github.com/Neue-Konzepte-BaaS/backend/internal/repositories/db"
	"github.com/Neue-Konzepte-BaaS/backend/internal/services"
	"github.com/google/uuid"
)

func TestFarmRepository_GetFarmByID(t *testing.T) {
	pool := setupTestDB(t)
	ctx := context.Background()

	farmRepo := repositories.NewFarmRepository(database.New(pool))

	t.Run("farmer with no fields or plots gets zero total, not an error", func(t *testing.T) {
		farmer, farmID, _, _ := seedFarmWithPlots(t, ctx, pool, 0)

		farm, err := farmRepo.GetFarmByID(ctx, farmID)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if farm.ID != farmID {
			t.Errorf("farm id = %v, want %v", farm.ID, farmID)
		}
		if farm.FarmerID != farmer {
			t.Errorf("farmer id = %v, want %v", farm.FarmerID, farmer)
		}
		if farm.Name != "Green Acres" {
			t.Errorf("name = %q, want Green Acres", farm.Name)
		}
		if farm.Address != "1 Farm Lane" {
			t.Errorf("address = %q, want '1 Farm Lane'", farm.Address)
		}
		if farm.TotalSquareMeters != 0 {
			t.Errorf("total square meters = %v, want 0", farm.TotalSquareMeters)
		}
		if farm.FoundedAt != nil {
			t.Errorf("founded at = %v, want nil", farm.FoundedAt)
		}
	})

	t.Run("total square meters sums plots across fields", func(t *testing.T) {
		_, farmID, _, _ := seedFarmWithPlots(t, ctx, pool, 3)

		farm, err := farmRepo.GetFarmByID(ctx, farmID)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if farm.TotalSquareMeters <= 0 {
			t.Errorf("total square meters = %v, want > 0", farm.TotalSquareMeters)
		}
	})

	t.Run("unknown farm id returns ErrNotFound", func(t *testing.T) {
		_, err := farmRepo.GetFarmByID(ctx, uuid.New())
		if err == nil {
			t.Fatal("expected an error for an unknown farm id")
		}
		if !errors.Is(err, services.ErrNotFound) {
			t.Errorf("err = %v, want ErrNotFound", err)
		}
	})
}

func TestFarmRepository_GetFarmIDByFarmerID(t *testing.T) {
	pool := setupTestDB(t)
	ctx := context.Background()

	farmRepo := repositories.NewFarmRepository(database.New(pool))

	t.Run("resolves the farmer's own farm id", func(t *testing.T) {
		farmer, farmID, _, _ := seedFarmWithPlots(t, ctx, pool, 0)

		got, err := farmRepo.GetFarmIDByFarmerID(ctx, farmer)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got != farmID {
			t.Errorf("farm id = %v, want %v", got, farmID)
		}
	})

	t.Run("unknown farmer id returns ErrNotFound", func(t *testing.T) {
		_, err := farmRepo.GetFarmIDByFarmerID(ctx, uuid.New())
		if err == nil {
			t.Fatal("expected an error for an unknown farmer id")
		}
		if !errors.Is(err, services.ErrNotFound) {
			t.Errorf("err = %v, want ErrNotFound", err)
		}
	})
}
