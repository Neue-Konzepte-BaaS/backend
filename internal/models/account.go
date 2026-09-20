package models

import (
	"time"

	"github.com/google/uuid"
)

// Role is implied by which subtype table an account joins to.
type Role string

const (
	RoleAdmin    Role = "admin"
	RoleFarmer   Role = "farmer"
	RoleCustomer Role = "customer"
)

type Account struct {
	ID           uuid.UUID
	FirstName    string
	LastName     string
	Email        string
	PasswordHash string
	Role         Role
	// PostalCode is 0 for an admin (no farmer/customer subtype row to carry
	// one) -- treat 0 as "unknown", not literally postal code zero, same
	// convention the frontend already applies to it.
	PostalCode int32
}

// AccountListing is one row of the admin account list.
//
// It is deliberately not models.Account: a listing must never be able to carry
// a password hash, so the field does not exist here rather than merely being
// left out of the JSON.
type AccountListing struct {
	ID        uuid.UUID
	FirstName string
	LastName  string
	Email     string
	// Role is empty for an account with no subtype row at all. That is an
	// orphan -- see the ELSE '' in GetAccountByEmail and the positive
	// membership test in GetAllRecipients -- and the admin list is the one
	// place it should be visible rather than quietly skipped.
	Role      Role
	CreatedAt time.Time
}

// AccountListFilter narrows the admin account list. The zero value lists
// everything; an empty Role or Query means "no filter" rather than "match
// nothing", which is what lets one query serve every combination.
type AccountListFilter struct {
	Role   Role
	Query  string
	Limit  int32
	Offset int32
}
