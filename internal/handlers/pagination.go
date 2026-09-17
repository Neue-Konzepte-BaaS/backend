package handlers

import (
	"errors"
	"strconv"

	"github.com/Neue-Konzepte-BaaS/backend/internal/services"
)

// The bounds live in services, which is where clampPagination applies them;
// the handler only enforces them earlier and more loudly.
const (
	defaultPageLimit = services.DefaultPageLimit
	maxPageLimit     = services.MaxPageLimit
)

var errInvalidPagination = errors.New("invalid pagination parameter")

// parseLimit reads a limit query parameter, falling back to def when it is
// absent. Anything present but outside [1, max] is an error rather than a
// silent clamp: a caller asking for 1000 rows should learn that it cannot have
// them instead of quietly receiving 100 and paging wrongly from there.
func parseLimit(raw string, def, max int32) (int32, error) {
	if raw == "" {
		return def, nil
	}
	limit, err := strconv.Atoi(raw)
	if err != nil || limit < 1 || limit > int(max) {
		return 0, errInvalidPagination
	}
	return int32(limit), nil
}

// parseOffset reads an offset query parameter, defaulting to 0. Negative
// offsets are rejected; there is no upper bound, an offset past the end simply
// returns an empty page.
func parseOffset(raw string) (int32, error) {
	if raw == "" {
		return 0, nil
	}
	offset, err := strconv.Atoi(raw)
	if err != nil || offset < 0 {
		return 0, errInvalidPagination
	}
	return int32(offset), nil
}
