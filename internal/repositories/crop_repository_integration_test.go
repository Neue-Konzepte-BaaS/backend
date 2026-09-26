package repositories_test

import (
	"context"
	"testing"

	"github.com/Neue-Konzepte-BaaS/backend/internal/repositories"
	database "github.com/Neue-Konzepte-BaaS/backend/internal/repositories/db"
	"github.com/google/uuid"
)

// TestGetAllCrops_HidesPlaceholderButKeepsItResolvable checks that the
// migration-seeded 'unknown' crop never shows up in the offerable catalog,
// while legacy rentals that point at it can still look it up by id.
func TestGetAllCrops_HidesPlaceholderButKeepsItResolvable(t *testing.T) {
	pool := setupTestDB(t)
	ctx := context.Background()

	cropRepo := repositories.NewCropRepository(pool, database.New(pool))

	var placeholderID uuid.UUID
	if err := pool.QueryRow(ctx, `SELECT id FROM crop WHERE name = 'unknown'`).Scan(&placeholderID); err != nil {
		t.Fatalf("looking up placeholder crop: %v", err)
	}

	crops, err := cropRepo.GetAllCrops(ctx)
	if err != nil {
		t.Fatalf("listing crops: %v", err)
	}
	for _, c := range crops {
		if c.ID == placeholderID {
			t.Errorf("GetAllCrops returned the placeholder crop %q", c.Name)
		}
	}

	crop, err := cropRepo.GetCropByID(ctx, placeholderID)
	if err != nil {
		t.Fatalf("GetCropByID(placeholder): %v", err)
	}
	if crop.Name != "unknown" {
		t.Errorf("placeholder name = %q, want %q", crop.Name, "unknown")
	}
}
