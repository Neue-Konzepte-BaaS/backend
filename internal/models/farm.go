package models

import (
	"time"

	"github.com/google/uuid"
)

type Farm struct {
	ID                uuid.UUID
	FarmerID          uuid.UUID
	Name              string
	Address           string
	Description       string
	FoundedAt         *time.Time
	TotalSquareMeters float64
}

// FarmUpdate is what a farmer may change about their own farm. The area is
// derived from the plots and the owner is fixed, so neither is here.
type FarmUpdate struct {
	Name        string
	Address     string
	Description string
	// FoundedAt nil clears the founding date.
	FoundedAt *time.Time
}

// FarmListing is one row of the admin farm list: a farm, the farmer account
// that owns it, and what it holds.
//
// Fields and Plots deliberately reuse the statistics types rather than
// declaring parallel ones. The figures are the same figures, computed by the
// same predicates, and reusing the types is what keeps them from drifting into
// two different meanings of "occupancy".
//
// The public farm view (Farm) and this one answer different questions, so they
// carry different columns: Farm has the description and founding date a visitor
// reads, this has the holdings and the owner an operator scans.
type FarmListing struct {
	ID       uuid.UUID
	FarmerID uuid.UUID
	Name     string
	Address  string
	// PostalCode lives on the farmer, not the farm -- the farm has a free-text
	// address instead, which is not something to filter on.
	PostalCode int32
	FirstName  string
	LastName   string
	Email      string
	// CreatedAt is when the owning account registered; farm has no timestamp
	// of its own.
	CreatedAt time.Time
	Fields    FieldStatistics
	Plots     PlotStatistics
	// ActiveRentals counts rentals on this farm's plots covering the present
	// moment. It can exceed Plots.Rented only if a plot were double-booked,
	// which the exclusion constraint forbids -- so in practice they agree, and
	// disagreeing would be a bug worth seeing.
	ActiveRentals int64
}

// FarmListFilter narrows the admin farm list. An empty Query or unset
// PostalCode means "no filter".
//
// There is deliberately no farm-id or farmer-id filter: the scope is the whole
// platform, so there is no identity parameter here for a future second caller
// of this endpoint to tamper with.
type FarmListFilter struct {
	Query      string
	PostalCode *int32
	Limit      int32
	Offset     int32
}
