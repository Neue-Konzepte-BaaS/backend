package repositories

import (
	"context"
	"errors"
	"fmt"

	database "github.com/Neue-Konzepte-BaaS/backend/internal/repositories/db"
	"github.com/Neue-Konzepte-BaaS/backend/internal/services"
	"github.com/jackc/pgx/v5"
	geom "github.com/twpayne/go-geom"
)

type postalCodeRepository struct {
	queries *database.Queries
}

func NewPostalCodeRepository(queries *database.Queries) services.PostalCodeRepository {
	return &postalCodeRepository{queries: queries}
}

func (r *postalCodeRepository) FindCoordinates(ctx context.Context, postalCode, city string) (float64, float64, error) {
	var point *geom.Point
	var err error

	if postalCode != "" {
		point, err = r.queries.FindPostalCodeCoordinates(ctx, postalCode)
	} else {
		point, err = r.queries.FindPostalCodeCoordinatesByCity(ctx, "%"+city+"%")
	}
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return 0, 0, fmt.Errorf("db error: %w %w", err, services.ErrNotFound)
		}
		return 0, 0, err
	}

	coords := point.Coords()
	return coords.X(), coords.Y(), nil
}
