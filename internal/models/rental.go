package models

import (
	"time"

	"github.com/google/uuid"
)

// RentalStatus is the state of a rental request: it starts Requested, and a
// farmer decides it into Approved or Declined exactly once.
type RentalStatus string

const (
	RentalStatusRequested RentalStatus = "requested"
	RentalStatusApproved  RentalStatus = "approved"
	RentalStatusDeclined  RentalStatus = "declined"
)

// Rental is a customer's request to book a plot for a half-open time period
// [StartAt, EndAt): a rental ending exactly when the next begins does not
// conflict with it. It only occupies the plot once Status is Approved; while
// Requested it still blocks the period so two customers can't both be
// approved for it, and once Declined it stops blocking entirely.
type Rental struct {
	ID        uuid.UUID
	PlotID    uuid.UUID
	CropID    uuid.UUID
	Customer  uuid.UUID
	StartAt   time.Time
	EndAt     time.Time
	Status    RentalStatus
	Message   string
	DecidedAt *time.Time
}

type RentalWithPlot struct {
	Rental
	Plot Plot
	Crop Crop
}

// RentalWithField pairs a rental with the id of the field its plot belongs
// to, for checking that a deciding farmer owns that field.
type RentalWithField struct {
	Rental
	Field uuid.UUID
}

type RentalWithPlotAndCustomer struct {
	Rental
	Plot      Plot
	FieldName string
	Customer  Recipient
}

// ActiveRental is a rental covering right now, with the plot, field and crop
// it books and where today falls inside its period. Both week numbers come
// from the database clock rather than the API host's — see the query in
// sql/queries/rental.sql.
type ActiveRental struct {
	Rental
	PlotName  string
	FieldName string
	// FarmID is the farm the plot belongs to, which decides whose version of
	// the crop's care guide the tenant reads.
	FarmID      uuid.UUID
	Crop        Crop
	CurrentWeek int32
	TotalWeeks  int32
}
