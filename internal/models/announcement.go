package models

import (
	"time"

	"github.com/google/uuid"
)

// Announcement is one notice a farmer posted to the Schwarzes Brett. It is
// kept after it has been mailed: the board is what a customer who missed the
// email reads instead.
type Announcement struct {
	ID        uuid.UUID
	Farmer    uuid.UUID
	Subject   string
	Body      string
	CreatedAt time.Time
}

// AnnouncementWithFarm is an announcement as a customer sees it. A customer
// rents from several farmers, so the board is only readable if each notice
// says which farm it came from.
type AnnouncementWithFarm struct {
	Announcement
	FarmName string
}
