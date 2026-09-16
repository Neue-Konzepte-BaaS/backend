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
