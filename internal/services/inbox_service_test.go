package services

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/Neue-Konzepte-BaaS/backend/internal/models"
	"github.com/google/uuid"
)

// fakeInboxBroadcastRepo answers the broadcast half of an inbox.
type fakeInboxBroadcastRepo struct {
	BroadcastNotificationRepository
	all []models.BroadcastNotification
	err error
}

func (f *fakeInboxBroadcastRepo) GetAllBroadcastNotifications(context.Context) ([]models.BroadcastNotification, error) {
	if f.err != nil {
		return nil, f.err
	}
	return f.all, nil
}

// fakeInboxAnnouncementRepo answers the announcement half of an inbox. Only
// GetAnnouncementsForCustomer is reachable from InboxService.
type fakeInboxAnnouncementRepo struct {
	AnnouncementRepository
	forCustomer []models.AnnouncementWithFarm
	err         error
}

func (f *fakeInboxAnnouncementRepo) GetAnnouncementsForCustomer(context.Context, uuid.UUID) ([]models.AnnouncementWithFarm, error) {
	if f.err != nil {
		return nil, f.err
	}
	return f.forCustomer, nil
}

func TestGetInboxForCustomer_MergesAndSortsNewestFirst(t *testing.T) {
	now := time.Now()

	broadcastRepo := &fakeInboxBroadcastRepo{all: []models.BroadcastNotification{
		{ID: uuid.New(), Subject: "Wartung", Body: "Sonntag offline", CreatedAt: now.Add(-2 * time.Hour)},
	}}
	announcementRepo := &fakeInboxAnnouncementRepo{forCustomer: []models.AnnouncementWithFarm{
		{
			Announcement: models.Announcement{ID: uuid.New(), Subject: "Ernte", Body: "Samstag um 9", CreatedAt: now},
			FarmName:     "Hof Grünwald",
		},
	}}
	svc := NewInboxService(broadcastRepo, announcementRepo)

	items, err := svc.GetInboxForCustomer(context.Background(), uuid.New())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(items) != 2 {
		t.Fatalf("got %d items, want 2", len(items))
	}
	if items[0].Kind != models.InboxItemAnnouncement || items[0].Subject != "Ernte" {
		t.Errorf("first item = %+v, want the newer announcement first", items[0])
	}
	if items[0].FarmName != "Hof Grünwald" {
		t.Errorf("first item farm name = %q, want it carried through", items[0].FarmName)
	}
	if items[1].Kind != models.InboxItemBroadcast || items[1].Subject != "Wartung" {
		t.Errorf("second item = %+v, want the older broadcast second", items[1])
	}
	if items[1].FarmName != "" {
		t.Errorf("broadcast farm name = %q, want empty", items[1].FarmName)
	}
}

func TestGetInboxForCustomer_BroadcastFailureIsReturned(t *testing.T) {
	boom := errors.New("db exploded")
	svc := NewInboxService(&fakeInboxBroadcastRepo{err: boom}, &fakeInboxAnnouncementRepo{})

	if _, err := svc.GetInboxForCustomer(context.Background(), uuid.New()); !errors.Is(err, boom) {
		t.Errorf("error = %v, want it to wrap the broadcast repository failure", err)
	}
}

func TestGetInboxForCustomer_AnnouncementFailureIsReturned(t *testing.T) {
	boom := errors.New("db exploded")
	svc := NewInboxService(&fakeInboxBroadcastRepo{}, &fakeInboxAnnouncementRepo{err: boom})

	if _, err := svc.GetInboxForCustomer(context.Background(), uuid.New()); !errors.Is(err, boom) {
		t.Errorf("error = %v, want it to wrap the announcement repository failure", err)
	}
}

func TestGetInboxForCustomer_EmptyIsNotAnError(t *testing.T) {
	svc := NewInboxService(&fakeInboxBroadcastRepo{}, &fakeInboxAnnouncementRepo{})

	items, err := svc.GetInboxForCustomer(context.Background(), uuid.New())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(items) != 0 {
		t.Errorf("got %d items, want 0", len(items))
	}
}
