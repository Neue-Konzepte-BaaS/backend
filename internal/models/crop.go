package models

import "github.com/google/uuid"

type Crop struct {
	ID             uuid.UUID
	Name           string
	DurationMonths int32
}
