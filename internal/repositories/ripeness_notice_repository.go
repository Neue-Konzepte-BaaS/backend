package repositories

import (
	"context"

	"github.com/Neue-Konzepte-BaaS/backend/internal/models"
	database "github.com/Neue-Konzepte-BaaS/backend/internal/repositories/db"
	"github.com/Neue-Konzepte-BaaS/backend/internal/services"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
)

type ripenessNoticeRepository struct {
	queries *database.Queries
}

func NewRipenessNoticeRepository(queries *database.Queries) services.RipenessNoticeRepository {
	return &ripenessNoticeRepository{queries: queries}
}

func (r *ripenessNoticeRepository) CreateRipenessNotice(ctx context.Context, farmer, field, crop uuid.UUID) (models.RipenessNoticeWithDetails, error) {
	row, err := r.queries.InsertRipenessNotice(ctx, database.InsertRipenessNoticeParams{
		Farmer: farmer,
		Field:  field,
		Crop:   crop,
	})
	if err != nil {
		// The caller already checked field ownership before reaching here, so
		// a FK violation at this point means an unknown crop id.
		return models.RipenessNoticeWithDetails{}, mapForeignKeyError(err)
	}

	return toModelRipenessNotice(row.ID, row.Farmer, row.Field, row.Crop, row.CreatedAt, row.FarmName, row.FieldName, row.CropName), nil
}

func (r *ripenessNoticeRepository) GetRipenessNoticesForCustomer(ctx context.Context, customer uuid.UUID) ([]models.RipenessNoticeWithDetails, error) {
	rows, err := r.queries.GetRipenessNoticesForCustomer(ctx, customer)
	if err != nil {
		return nil, err
	}

	notices := make([]models.RipenessNoticeWithDetails, len(rows))
	for i, row := range rows {
		notices[i] = toModelRipenessNotice(row.ID, row.Farmer, row.Field, row.Crop, row.CreatedAt, row.FarmName, row.FieldName, row.CropName)
	}
	return notices, nil
}

func toModelRipenessNotice(id, farmer, field, crop uuid.UUID, createdAt pgtype.Timestamptz, farmName, fieldName, cropName string) models.RipenessNoticeWithDetails {
	return models.RipenessNoticeWithDetails{
		RipenessNotice: models.RipenessNotice{
			ID:        id,
			Farmer:    farmer,
			Field:     field,
			Crop:      crop,
			CreatedAt: createdAt.Time,
		},
		FarmName:  farmName,
		FieldName: fieldName,
		CropName:  cropName,
	}
}
