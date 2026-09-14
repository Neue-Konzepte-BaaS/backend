package repositories

import (
	"context"

	"github.com/Neue-Konzepte-BaaS/backend/internal/models"
	database "github.com/Neue-Konzepte-BaaS/backend/internal/repositories/db"
	"github.com/Neue-Konzepte-BaaS/backend/internal/services"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
)

type announcementRepository struct {
	queries *database.Queries
}

func NewAnnouncementRepository(queries *database.Queries) services.AnnouncementRepository {
	return &announcementRepository{queries: queries}
}

func (r *announcementRepository) CreateAnnouncement(ctx context.Context, farmer uuid.UUID, subject, body string) (models.AnnouncementWithFarm, error) {
	row, err := r.queries.InsertAnnouncement(ctx, database.InsertAnnouncementParams{
		Farmer:  farmer,
		Subject: subject,
		Body:    body,
	})
	if err != nil {
		return models.AnnouncementWithFarm{}, err
	}

	return toModelAnnouncement(row.ID, row.Farmer, row.Subject, row.Body, row.CreatedAt, row.FarmName), nil
}

func (r *announcementRepository) GetAnnouncementsByFarmer(ctx context.Context, farmer uuid.UUID) ([]models.AnnouncementWithFarm, error) {
	rows, err := r.queries.GetAnnouncementsByFarmer(ctx, farmer)
	if err != nil {
		return nil, err
	}

	announcements := make([]models.AnnouncementWithFarm, len(rows))
	for i, row := range rows {
		announcements[i] = toModelAnnouncement(row.ID, row.Farmer, row.Subject, row.Body, row.CreatedAt, row.FarmName)
	}
	return announcements, nil
}

func (r *announcementRepository) GetAnnouncementsForCustomer(ctx context.Context, customer uuid.UUID) ([]models.AnnouncementWithFarm, error) {
	rows, err := r.queries.GetAnnouncementsForCustomer(ctx, customer)
	if err != nil {
		return nil, err
	}

	announcements := make([]models.AnnouncementWithFarm, len(rows))
	for i, row := range rows {
		announcements[i] = toModelAnnouncement(row.ID, row.Farmer, row.Subject, row.Body, row.CreatedAt, row.FarmName)
	}
	return announcements, nil
}

// toModelAnnouncement maps one announcement row to the domain model. The three
// queries return identical but distinct generated row types, so the fields are
// passed individually rather than the row itself.
func toModelAnnouncement(id, farmer uuid.UUID, subject, body string, createdAt pgtype.Timestamptz, farmName string) models.AnnouncementWithFarm {
	return models.AnnouncementWithFarm{
		Announcement: models.Announcement{
			ID:        id,
			Farmer:    farmer,
			Subject:   subject,
			Body:      body,
			CreatedAt: createdAt.Time,
		},
		FarmName: farmName,
	}
}
