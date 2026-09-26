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
// The guide belongs to the crop, not to a farm: the catalog is admin-owned
// (only an admin may add a crop), and so is the advice attached to it.
type CareInstruction struct {
	ID        uuid.UUID
	Crop      uuid.UUID
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
