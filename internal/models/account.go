package models

import "github.com/google/uuid"

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
}
