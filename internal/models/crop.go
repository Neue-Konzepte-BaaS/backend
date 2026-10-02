package models

import "github.com/google/uuid"

type Crop struct {
	ID             uuid.UUID
	NameDe         string
	NameEn         string
	DurationMonths int32
}
