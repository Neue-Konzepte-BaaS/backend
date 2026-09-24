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

// exclusionViolation is the SQLSTATE for an EXCLUDE constraint breach; on the
// rental table that means the plot is already rented for an overlapping period.
const exclusionViolation = "23P01"

// foreignKeyViolation is the SQLSTATE for a missing referenced row, e.g.
// renting a plot id that does not exist.
const foreignKeyViolation = "23503"

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

// mapForeignKeyError turns a foreign key violation — inserting a row that
// references an id which does not exist — into services.ErrNotFound, so
// handlers can report it as a 404 instead of leaking a raw SQL error as a
// 500. Shared by every repository that inserts a row referencing an id the
// caller supplied rather than one it just looked up itself: crop_repository
// (an unknown crop or plot id), announcement_repository and
// ripeness_notice_repository (an unknown field, plot or crop id).
func mapForeignKeyError(err error) error {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == foreignKeyViolation {
		return fmt.Errorf("%s: %w", pgErr.Message, services.ErrNotFound)
	}
	return err
}
