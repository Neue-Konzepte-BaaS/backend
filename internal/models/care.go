package models

import (
	"time"

	"github.com/google/uuid"
)

// CareInstruction is one task in a crop's weekly care guide: what a tenant
// should do in a given week of their rental. Week 1 is the first seven days
// after the plot was booked, not a calendar week — see the table comment in
// sql/migrations/20260921150000_care_instructions.sql.
//
// Every crop has a default guide, which an admin maintains and every farm
// starts from. A farmer may take a crop's guide over for their own farm; from
// then on that farm's tenants read the farm's version instead — see
// sql/migrations/20260925100000_farm_care_guides.sql.
type CareInstruction struct {
	ID   uuid.UUID
	Crop uuid.UUID
	// Farm is nil for a step of the default guide, and the farm's id for a
	// step of that farm's own guide.
	Farm      *uuid.UUID
	Week      int32
	Title     string
	Body      string
	CreatedAt time.Time
	UpdatedAt time.Time
}

// PlotCareGuide is one rented plot's care guide as its tenant sees it: the
// crop's instructions, plus where today falls inside that rental so the
// client can tell this week's tasks from the ones still ahead.
type PlotCareGuide struct {
	RentalID  uuid.UUID
	PlotID    uuid.UUID
	PlotName  string
	FieldName string
	Crop      Crop
	StartAt   time.Time
	EndAt     time.Time
	// CurrentWeek is the week of the rental today falls in, counting from 1.
	CurrentWeek int32
	// TotalWeeks is how many weeks the rental runs, rounded up.
	TotalWeeks int32
	// Instructions are the crop's tasks in week order, limited to the weeks
	// this rental actually reaches.
	Instructions []CareInstruction
}

// CropAtFarm names one guide a tenant can read: a crop as grown on one farm.
// Which version that is — the farm's own or the default — is the repository's
// to resolve.
type CropAtFarm struct {
	Crop uuid.UUID
	Farm uuid.UUID
}

// CropCareGuide is one crop's guide as an editor sees it: the default for an
// admin, and for a farmer whichever version their tenants read.
type CropCareGuide struct {
	// FarmGuide is true when the instructions are the farmer's own version
	// rather than the default. Separate from the instructions because a
	// farm's own guide may be empty.
	FarmGuide    bool
	Instructions []CareInstruction
}
