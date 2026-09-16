package models

import (
	"time"

	"github.com/google/uuid"
)

// FarmListing is one row of the admin farm list: a farmer's account, the farm
// it owns, and what that farm holds.
//
// There is no farm table -- a farm *is* a farmer subtype row -- so Account is
// both the owner's id and the farm's.
//
// Fields and Plots deliberately reuse the statistics types rather than
// declaring parallel ones. The figures are the same figures, computed by the
// same predicates, and reusing the types is what keeps them from drifting into
// two different meanings of "occupancy".
type FarmListing struct {
	Account    uuid.UUID
	FarmName   string
	PostalCode int32
	FirstName  string
	LastName   string
	Email      string
	CreatedAt  time.Time
	Fields     FieldStatistics
	Plots      PlotStatistics
	// ActiveRentals counts rentals on this farm's plots covering the present
	// moment. It can exceed Plots.Rented only if a plot were double-booked,
	// which the exclusion constraint forbids -- so in practice they agree, and
	// disagreeing would be a bug worth seeing.
	ActiveRentals int64
}

// FarmListFilter narrows the admin farm list. As with AccountListFilter, an
// empty Query or unset PostalCode means "no filter".
//
// There is deliberately no farmer-id filter: an admin has no use for one, and
// leaving it out means there is no identity parameter here for a future second
// caller of this endpoint to tamper with.
type FarmListFilter struct {
	Query      string
	PostalCode *int32
	Limit      int32
	Offset     int32
}
