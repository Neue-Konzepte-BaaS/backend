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

// TestGetAllCrops_HidesPlaceholderButKeepsItResolvable checks that the
// migration-seeded 'unknown' crop never shows up in the offerable catalog,
// while legacy rentals that point at it can still look it up by id.
func TestGetAllCrops_HidesPlaceholderButKeepsItResolvable(t *testing.T) {
	pool := setupTestDB(t)
	ctx := context.Background()

	cropRepo := repositories.NewCropRepository(pool, database.New(pool))

	var placeholderID uuid.UUID
	if err := pool.QueryRow(ctx, `SELECT id FROM crop WHERE name_de = 'unknown'`).Scan(&placeholderID); err != nil {
		t.Fatalf("looking up placeholder crop: %v", err)
	}

	crops, err := cropRepo.GetAllCrops(ctx)
	if err != nil {
		t.Fatalf("listing crops: %v", err)
	}
	for _, c := range crops {
		if c.ID == placeholderID {
			t.Errorf("GetAllCrops returned the placeholder crop %q", c.NameDe)
		}
	}

	crop, err := cropRepo.GetCropByID(ctx, placeholderID)
	if err != nil {
		t.Fatalf("GetCropByID(placeholder): %v", err)
	}
	if crop.NameDe != "unknown" {
		t.Errorf("placeholder name = %q, want %q", crop.NameDe, "unknown")
	}
}

// TestCreateCrop_NameUniquenessIsPerLanguage checks that nameDe and nameEn
// are each independently unique in the catalog: a new crop must be rejected
// if it collides with an existing crop on either language, not just when
// both collide at once.
func TestCreateCrop_NameUniquenessIsPerLanguage(t *testing.T) {
	pool := setupTestDB(t)
	ctx := context.Background()

	cropRepo := repositories.NewCropRepository(pool, database.New(pool))

	base := uuid.NewString()
	nameDe := "Kartoffel-" + base
	nameEn := "Potato-" + base

	if _, err := cropRepo.CreateCrop(ctx, nameDe, nameEn, 3); err != nil {
		t.Fatalf("creating base crop: %v", err)
	}

	if _, err := cropRepo.CreateCrop(ctx, nameDe, "Unique-"+uuid.NewString(), 3); !errors.Is(err, services.ErrCropNameTaken) {
		t.Errorf("colliding nameDe: err = %v, want ErrCropNameTaken", err)
	}

	if _, err := cropRepo.CreateCrop(ctx, "Unique-"+uuid.NewString(), nameEn, 3); !errors.Is(err, services.ErrCropNameTaken) {
		t.Errorf("colliding nameEn: err = %v, want ErrCropNameTaken", err)
	}
}
