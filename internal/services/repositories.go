package services

import (
	"context"

	"github.com/Neue-Konzepte-BaaS/backend/internal/models"
	"github.com/google/uuid"
)

type AccountRepository interface {
	GetAccountByEmail(ctx context.Context, email string) (models.Account, error)
	GetAccountByID(ctx context.Context, id uuid.UUID) (models.Account, error)
}

type FieldRepository interface {
	CreateField(ctx context.Context, field models.Field) (uuid.UUID, error)
	GetFieldOwner(ctx context.Context, id uuid.UUID) (uuid.UUID, error)
	GetFieldsByFarmer(ctx context.Context, farmer uuid.UUID) ([]models.Field, error)
}

type PlotRepository interface {
	CreatePlot(ctx context.Context, plot models.Plot) (uuid.UUID, error)
	GetPlotsByFields(ctx context.Context, fields []uuid.UUID) ([]models.Plot, error)
}
