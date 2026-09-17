package models

import (
	"time"

	"github.com/google/uuid"
)

// InboxItemKind distinguishes what an inbox entry originated from.
type InboxItemKind string

const (
	InboxItemBroadcast    InboxItemKind = "broadcast"
	InboxItemAnnouncement InboxItemKind = "announcement"
)

// InboxItem is one entry in a customer's merged inbox: a broadcast
// notification or an announcement from a farmer they currently rent from,
// normalized to a common shape so the client renders one feed instead of two.
type InboxItem struct {
	Kind      InboxItemKind
	ID        uuid.UUID
	Subject   string
	Body      string
	FarmName  string // empty for broadcasts
	CreatedAt time.Time
}
