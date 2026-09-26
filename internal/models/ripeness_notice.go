package models

import (
	"time"

	"github.com/google/uuid"
)

// RipenessNotice is a farmer's notice that a crop is ready to harvest on one
// of his fields.
type RipenessNotice struct {
	ID        uuid.UUID
	Farmer    uuid.UUID
	Field     uuid.UUID
	Crop      uuid.UUID
	CreatedAt time.Time
}

// RipenessNoticeWithDetails is a ripeness notice as a customer or farmer sees
// it, with the farm, field and crop names already resolved.
type RipenessNoticeWithDetails struct {
	RipenessNotice
	FarmName  string
	FieldName string
	CropName  string
}
