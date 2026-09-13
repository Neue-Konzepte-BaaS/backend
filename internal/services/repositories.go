package services

import (
	"context"

	"github.com/Neue-Konzepte-BaaS/backend/internal/models"
	"github.com/google/uuid"
)

type AccountRepository interface {
	GetAccountByEmail(ctx context.Context, email string) (models.Account, error)
	GetAccountByID(ctx context.Context, id uuid.UUID) (models.Account, error)
	// CreateFarmer atomically inserts the account and its farmer subtype row.
	CreateFarmer(ctx context.Context, account models.Account, farmName string, postalCode int32) (models.Account, error)
	// CreateCustomer atomically inserts the account and its customer subtype row.
	CreateCustomer(ctx context.Context, account models.Account, postalCode int32) (models.Account, error)
}
