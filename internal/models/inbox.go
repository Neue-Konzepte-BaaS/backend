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
)

// InboxItem is one entry in a customer's merged inbox: a broadcast
// notification, an announcement from a farmer they currently rent from, or a
// ripeness notice for a crop they are growing — normalized to a common shape
// so the client renders one feed instead of three.
type InboxItem struct {
	Kind      InboxItemKind
	ID        uuid.UUID
	Subject   string
	Body      string
	FarmName  string // empty for broadcasts
	FieldName string // set only for ripeness notices
	CropName  string // set only for ripeness notices
	CreatedAt time.Time
}
