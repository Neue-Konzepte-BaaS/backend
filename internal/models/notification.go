package models

import (
	"strings"

	"github.com/google/uuid"
)

// Recipient is an account reduced to what a notification channel needs to
// reach it. Notifications are addressed to people rather than to accounts, so
// nothing role- or auth-related belongs here.
type Recipient struct {
	AccountID uuid.UUID
	Email     string
	FirstName string
	LastName  string
}

// DisplayName is the name a notification addresses the recipient by. The
// account table has no display-name column, so it is built from the two name
// parts; either may be blank in principle, hence the trim.
func (r Recipient) DisplayName() string {
	return strings.TrimSpace(r.FirstName + " " + r.LastName)
}
