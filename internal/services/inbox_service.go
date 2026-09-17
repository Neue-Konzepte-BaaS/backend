package services

import (
	"context"
	"fmt"
	"sort"

	"github.com/Neue-Konzepte-BaaS/backend/internal/models"
	"github.com/google/uuid"
)

// InboxService merges the messages a customer can see: every platform-wide
// broadcast, and the announcements of every farmer he currently rents from.
type InboxService interface {
	// GetInboxForCustomer returns the customer's merged inbox, newest first.
	GetInboxForCustomer(ctx context.Context, customer uuid.UUID) ([]models.InboxItem, error)
}

type inboxService struct {
	broadcastRepo    BroadcastNotificationRepository
	announcementRepo AnnouncementRepository
}

func NewInboxService(broadcastRepo BroadcastNotificationRepository, announcementRepo AnnouncementRepository) InboxService {
	return &inboxService{
		broadcastRepo:    broadcastRepo,
		announcementRepo: announcementRepo,
	}
}

func (s *inboxService) GetInboxForCustomer(ctx context.Context, customer uuid.UUID) ([]models.InboxItem, error) {
	broadcasts, err := s.broadcastRepo.GetAllBroadcastNotifications(ctx)
	if err != nil {
		return nil, fmt.Errorf("getting broadcast notifications: %w", err)
	}

	announcements, err := s.announcementRepo.GetAnnouncementsForCustomer(ctx, customer)
	if err != nil {
		return nil, fmt.Errorf("getting announcements: %w", err)
	}

	items := make([]models.InboxItem, 0, len(broadcasts)+len(announcements))
	for _, b := range broadcasts {
		items = append(items, models.InboxItem{
			Kind:      models.InboxItemBroadcast,
			ID:        b.ID,
			Subject:   b.Subject,
			Body:      b.Body,
			CreatedAt: b.CreatedAt,
		})
	}
	for _, a := range announcements {
		items = append(items, models.InboxItem{
			Kind:      models.InboxItemAnnouncement,
			ID:        a.ID,
			Subject:   a.Subject,
			Body:      a.Body,
			FarmName:  a.FarmName,
			CreatedAt: a.CreatedAt,
		})
	}

	sort.Slice(items, func(i, j int) bool {
		return items[i].CreatedAt.After(items[j].CreatedAt)
	})

	return items, nil
}
