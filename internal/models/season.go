package models

import (
	"time"

	"github.com/google/uuid"
)

// Season is a recurring yearly calendar window - "Spring: Mar 1 - May 31" -
// that repeats every year regardless of which specific year a rental falls
// in. Farm is nil for one of the global defaults every farm starts from, and
// the farm's id for that farm's own version, which a farmer may create,
// edit, and delete freely - see sql/migrations/20260926090000_seasons.sql.
type Season struct {
	ID                   uuid.UUID
	Farm                 *uuid.UUID
	Name                 string
	StartMonth, StartDay int32
	EndMonth, EndDay     int32
	CreatedAt, UpdatedAt time.Time
}

// CropSeasonRule ties a crop to a season, so renting it is only allowed
// inside that season's window. Farm is nil for the default rule an admin
// sets, applying to every farm that has not overridden it, or the farm's id
// for that farm's own rule - whose Season must belong to that same farm (see
// sql/migrations/20260926090100_crop_season.sql). A crop with no rule at
// all, for a farm or as a default, is unrestricted / year-round.
type CropSeasonRule struct {
	ID     uuid.UUID
	Crop   uuid.UUID
	Season uuid.UUID
	Farm   *uuid.UUID
}

// CropWithSeason is a crop from the catalog together with the season it is
// effectively checked against for the caller: the default rule for an admin,
// or the effective rule (their own farm's if it has one, the default
// otherwise) for a farmer. Nil means the crop is unrestricted for them.
type CropWithSeason struct {
	Crop   Crop
	Season *Season
}

// Contains reports whether the given month/day falls within the season,
// handling a season that wraps the new year (e.g. Winter: Dec 1 - Feb 28) by
// treating start > end as "wraps": in season from start to Dec 31, and from
// Jan 1 to end.
func (s Season) Contains(month, day int32) bool {
	key := month*100 + day
	start := s.StartMonth*100 + s.StartDay
	end := s.EndMonth*100 + s.EndDay
	if start <= end {
		return key >= start && key <= end
	}
	return key >= start || key <= end
}
