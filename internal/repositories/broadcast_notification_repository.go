package repositories

import (
	"context"

	"github.com/Neue-Konzepte-BaaS/backend/internal/models"
	database "github.com/Neue-Konzepte-BaaS/backend/internal/repositories/db"
	"github.com/Neue-Konzepte-BaaS/backend/internal/services"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
)

type broadcastNotificationRepository struct {
	queries *database.Queries
}

func NewBroadcastNotificationRepository(queries *database.Queries) services.BroadcastNotificationRepository {
	return &broadcastNotificationRepository{queries: queries}
}

func (r *broadcastNotificationRepository) CreateBroadcastNotification(ctx context.Context, subject, body string) (models.BroadcastNotification, error) {
	row, err := r.queries.InsertBroadcastNotification(ctx, database.InsertBroadcastNotificationParams{
		Subject: subject,
		Body:    body,
	})
	if err != nil {
		return models.BroadcastNotification{}, err
	}
	return toModelBroadcastNotification(row.ID, row.Subject, row.Body, row.CreatedAt), nil
}

func (r *broadcastNotificationRepository) GetAllBroadcastNotifications(ctx context.Context) ([]models.BroadcastNotification, error) {
	rows, err := r.queries.GetAllBroadcastNotifications(ctx)
	if err != nil {
		return nil, err
	}

	notifications := make([]models.BroadcastNotification, len(rows))
	for i, row := range rows {
		notifications[i] = toModelBroadcastNotification(row.ID, row.Subject, row.Body, row.CreatedAt)
	}
	return notifications, nil
}

func toModelBroadcastNotification(id uuid.UUID, subject, body string, createdAt pgtype.Timestamptz) models.BroadcastNotification {
	return models.BroadcastNotification{
		ID:        id,
		Subject:   subject,
		Body:      body,
		CreatedAt: createdAt.Time,
	}
}
