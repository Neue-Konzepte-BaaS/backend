package models

import (
	"time"

	"github.com/google/uuid"
)

// InboxItemKind distinguishes what an inbox entry originated from.
type InboxItemKind string

const (
	InboxItemBroadcast      InboxItemKind = "broadcast"
	InboxItemAnnouncement   InboxItemKind = "announcement"
	InboxItemRipenessNotice InboxItemKind = "ripeness_notice"
	InboxItemCare           InboxItemKind = "care"
)

// InboxItem is one entry in a customer's merged inbox: a broadcast
// notification, an announcement from a farmer they currently rent from, a
// ripeness notice for a crop they are growing, or a care instruction for a
// week of their rental that has already begun — normalized to a common shape
// so the client renders one feed instead of four.
type InboxItem struct {
	Kind      InboxItemKind
	ID        uuid.UUID
	Subject   string
	Body      string
	FarmName  string // empty for broadcasts and care instructions
	FieldName string // set only for ripeness notices and care instructions
	CropName  string // set only for ripeness notices and care instructions
	CreatedAt time.Time
}
