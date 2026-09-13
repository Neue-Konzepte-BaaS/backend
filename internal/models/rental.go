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
