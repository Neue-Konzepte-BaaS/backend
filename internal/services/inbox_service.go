package services

import (
	"context"
	"fmt"
	"sort"
	"time"

	"github.com/Neue-Konzepte-BaaS/backend/internal/models"
	"github.com/google/uuid"
)

// InboxService merges the messages a customer can see: every platform-wide
// broadcast, the announcements of every farmer he currently rents from, the
// ripeness notices for the crops he is currently growing, and the care
// instructions for every week of his current rentals that has already begun.
type InboxService interface {
	// GetInboxForCustomer returns the customer's merged inbox, newest first.
	GetInboxForCustomer(ctx context.Context, customer uuid.UUID) ([]models.InboxItem, error)
}

type inboxService struct {
	broadcastRepo      BroadcastNotificationRepository
	announcementRepo   AnnouncementRepository
	ripenessNoticeRepo RipenessNoticeRepository
	careGuideService   CareGuideService
}

func NewInboxService(broadcastRepo BroadcastNotificationRepository, announcementRepo AnnouncementRepository, ripenessNoticeRepo RipenessNoticeRepository, careGuideService CareGuideService) InboxService {
	return &inboxService{
		broadcastRepo:      broadcastRepo,
		announcementRepo:   announcementRepo,
		ripenessNoticeRepo: ripenessNoticeRepo,
		careGuideService:   careGuideService,
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

	ripenessNotices, err := s.ripenessNoticeRepo.GetRipenessNoticesForCustomer(ctx, customer)
	if err != nil {
		return nil, fmt.Errorf("getting ripeness notices: %w", err)
	}

	// The care guide already decides which rentals count (approved, covering
	// today) and drops the weeks a rental never reaches, so the inbox reuses
	// it rather than re-deriving either rule.
	careGuides, err := s.careGuideService.GetCareGuideForCustomer(ctx, customer)
	if err != nil {
		return nil, fmt.Errorf("getting care guide: %w", err)
	}

	items := make([]models.InboxItem, 0, len(broadcasts)+len(announcements)+len(ripenessNotices))
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
	for _, rn := range ripenessNotices {
		items = append(items, models.InboxItem{
			Kind:      models.InboxItemRipenessNotice,
			ID:        rn.ID,
			Subject:   fmt.Sprintf("%s ist reif", rn.CropName),
			Body:      fmt.Sprintf("%s auf %s ist bereit zur Ernte.", rn.CropName, rn.FieldName),
			FarmName:  rn.FarmName,
			FieldName: rn.FieldName,
			CropName:  rn.CropName,
			CreatedAt: rn.CreatedAt,
		})
	}

	for _, guide := range careGuides {
		items = append(items, careInboxItems(guide)...)
	}

	sort.Slice(items, func(i, j int) bool {
		return items[i].CreatedAt.After(items[j].CreatedAt)
	})

	return items, nil
}

// careInboxItems turns one rented plot's care guide into inbox items: one per
// instruction whose week has already begun. Upcoming weeks stay out — the
// inbox is a history of what the tenant was told, and a task for week 9 is not
// news in week 2; the care guide itself shows what lies ahead.
//
// A care instruction is never "sent", so its CreatedAt is the moment its week
// of this rental started. That puts this week's tasks near the top of the feed
// and last week's below them, in the order the tenant met them.
func careInboxItems(guide models.PlotCareGuide) []models.InboxItem {
	items := make([]models.InboxItem, 0, len(guide.Instructions))
	for _, instruction := range guide.Instructions {
		if instruction.Week > guide.CurrentWeek {
			continue
		}
		items = append(items, models.InboxItem{
			Kind: models.InboxItemCare,
			// The instruction belongs to the crop, so two plots growing the
			// same crop share it. Deriving the id from the rental keeps every
			// inbox item's id unique, and stable across reads.
			ID:        uuid.NewSHA1(guide.RentalID, instruction.ID[:]),
			Subject:   fmt.Sprintf("Woche %d: %s", instruction.Week, instruction.Title),
			Body:      instruction.Body,
			FieldName: guide.FieldName,
			CropName:  guide.Crop.Name,
			CreatedAt: guide.StartAt.Add(time.Duration(instruction.Week-1) * 7 * 24 * time.Hour),
		})
	}
	return items
}
