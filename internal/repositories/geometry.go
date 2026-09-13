package repositories

import (
	"errors"
	"fmt"

	"github.com/Neue-Konzepte-BaaS/backend/internal/services"
	"github.com/jackc/pgx/v5/pgconn"
)

// checkViolation is the SQLSTATE Postgres raises for CHECK constraint
// failures, e.g. the field/plot "must be a rectangle" constraint.
const checkViolation = "23514"

// raiseException is the SQLSTATE for a plain RAISE EXCEPTION in a trigger,
// which is how the plot-within-field boundary is enforced.
const raiseException = "P0001"

// mapGeometryError turns the Postgres errors produced by the field/plot
// geometry constraints into services.ErrInvalidGeometry, so handlers can
// report them as a 400 instead of leaking a raw SQL error as a 500.
func mapGeometryError(err error) error {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && (pgErr.Code == checkViolation || pgErr.Code == raiseException) {
		return fmt.Errorf("%s: %w", pgErr.Message, services.ErrInvalidGeometry)
	}
	return err
}
