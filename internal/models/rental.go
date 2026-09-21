package models

import (
	"time"

	"github.com/google/uuid"
)

// Rental is a customer's booking of a plot for a half-open time period
// [StartAt, EndAt): a rental ending exactly when the next begins does not
// conflict with it.
type Rental struct {
	ID       uuid.UUID
	PlotID   uuid.UUID
	CropID   uuid.UUID
	Customer uuid.UUID
	StartAt  time.Time
	EndAt    time.Time
}

type RentalWithPlot struct {
	Rental
	Plot Plot
	Crop Crop
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
	PlotName    string
	FieldName   string
	Crop        Crop
	CurrentWeek int32
	TotalWeeks  int32
}
