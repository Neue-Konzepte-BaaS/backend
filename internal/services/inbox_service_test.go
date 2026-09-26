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

// fakeInboxRipenessNoticeRepo answers the ripeness half of an inbox. Only
// GetRipenessNoticesForCustomer is reachable from InboxService.
type fakeInboxRipenessNoticeRepo struct {
	RipenessNoticeRepository
	forCustomer []models.RipenessNoticeWithDetails
	err         error
}

func (f *fakeInboxRipenessNoticeRepo) GetRipenessNoticesForCustomer(context.Context, uuid.UUID) ([]models.RipenessNoticeWithDetails, error) {
	if f.err != nil {
		return nil, f.err
	}
	return f.forCustomer, nil
}

// fakeInboxCareGuideService answers the care half of an inbox. Only
// GetCareGuideForCustomer is reachable from InboxService.
type fakeInboxCareGuideService struct {
	CareGuideService
	forCustomer []models.PlotCareGuide
	err         error
}

func (f *fakeInboxCareGuideService) GetCareGuideForCustomer(context.Context, uuid.UUID) ([]models.PlotCareGuide, error) {
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
	ripenessRepo := &fakeInboxRipenessNoticeRepo{forCustomer: []models.RipenessNoticeWithDetails{
		{
			RipenessNotice: models.RipenessNotice{ID: uuid.New(), CreatedAt: now.Add(-1 * time.Hour)},
			FarmName:       "Hof Grünwald",
			FieldName:      "Feld Nord",
			CropName:       "Zucchini",
		},
	}}
	svc := NewInboxService(broadcastRepo, announcementRepo, ripenessRepo, &fakeInboxCareGuideService{})

	items, err := svc.GetInboxForCustomer(context.Background(), uuid.New())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(items) != 3 {
		t.Fatalf("got %d items, want 3", len(items))
	}
	if items[0].Kind != models.InboxItemAnnouncement || items[0].Subject != "Ernte" {
		t.Errorf("first item = %+v, want the newest announcement first", items[0])
	}
	if items[0].FarmName != "Hof Grünwald" {
		t.Errorf("first item farm name = %q, want it carried through", items[0].FarmName)
	}
	if items[1].Kind != models.InboxItemRipenessNotice {
		t.Errorf("second item = %+v, want the ripeness notice second", items[1])
	}
	if items[1].FieldName != "Feld Nord" || items[1].CropName != "Zucchini" {
		t.Errorf("second item = %+v, want field and crop carried through", items[1])
	}
	if items[2].Kind != models.InboxItemBroadcast || items[2].Subject != "Wartung" {
		t.Errorf("third item = %+v, want the oldest broadcast last", items[2])
	}
	if items[2].FarmName != "" {
		t.Errorf("broadcast farm name = %q, want empty", items[2].FarmName)
	}
}

func TestGetInboxForCustomer_BroadcastFailureIsReturned(t *testing.T) {
	boom := errors.New("db exploded")
	svc := NewInboxService(&fakeInboxBroadcastRepo{err: boom}, &fakeInboxAnnouncementRepo{}, &fakeInboxRipenessNoticeRepo{}, &fakeInboxCareGuideService{})

	if _, err := svc.GetInboxForCustomer(context.Background(), uuid.New()); !errors.Is(err, boom) {
		t.Errorf("error = %v, want it to wrap the broadcast repository failure", err)
	}
}

func TestGetInboxForCustomer_AnnouncementFailureIsReturned(t *testing.T) {
	boom := errors.New("db exploded")
	svc := NewInboxService(&fakeInboxBroadcastRepo{}, &fakeInboxAnnouncementRepo{err: boom}, &fakeInboxRipenessNoticeRepo{}, &fakeInboxCareGuideService{})

	if _, err := svc.GetInboxForCustomer(context.Background(), uuid.New()); !errors.Is(err, boom) {
		t.Errorf("error = %v, want it to wrap the announcement repository failure", err)
	}
}

func TestGetInboxForCustomer_RipenessNoticeFailureIsReturned(t *testing.T) {
	boom := errors.New("db exploded")
	svc := NewInboxService(&fakeInboxBroadcastRepo{}, &fakeInboxAnnouncementRepo{}, &fakeInboxRipenessNoticeRepo{err: boom}, &fakeInboxCareGuideService{})

	if _, err := svc.GetInboxForCustomer(context.Background(), uuid.New()); !errors.Is(err, boom) {
		t.Errorf("error = %v, want it to wrap the ripeness notice repository failure", err)
	}
}

func TestGetInboxForCustomer_EmptyIsNotAnError(t *testing.T) {
	svc := NewInboxService(&fakeInboxBroadcastRepo{}, &fakeInboxAnnouncementRepo{}, &fakeInboxRipenessNoticeRepo{}, &fakeInboxCareGuideService{})

	items, err := svc.GetInboxForCustomer(context.Background(), uuid.New())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(items) != 0 {
		t.Errorf("got %d items, want 0", len(items))
	}
}

func TestGetInboxForCustomer_CareGuideFailureIsReturned(t *testing.T) {
	boom := errors.New("db exploded")
	svc := NewInboxService(&fakeInboxBroadcastRepo{}, &fakeInboxAnnouncementRepo{}, &fakeInboxRipenessNoticeRepo{}, &fakeInboxCareGuideService{err: boom})

	if _, err := svc.GetInboxForCustomer(context.Background(), uuid.New()); !errors.Is(err, boom) {
		t.Errorf("error = %v, want it to wrap the care guide failure", err)
	}
}

func TestGetInboxForCustomer_CareItemsForBegunWeeksOnly(t *testing.T) {
	startAt := time.Date(2026, 9, 1, 8, 0, 0, 0, time.UTC)
	week1 := models.CareInstruction{ID: uuid.New(), Week: 1, Title: "Aussäen", Body: "Reihen im Abstand von 30 cm"}
	week3 := models.CareInstruction{ID: uuid.New(), Week: 3, Title: "Gießen", Body: "Zweimal pro Woche"}
	week4 := models.CareInstruction{ID: uuid.New(), Week: 4, Title: "Mulchen", Body: "Stroh auslegen"}
	guide := models.PlotCareGuide{
		RentalID:     uuid.New(),
		FieldName:    "Feld Nord",
		Crop:         models.Crop{Name: "Zucchini"},
		StartAt:      startAt,
		CurrentWeek:  3,
		TotalWeeks:   13,
		Instructions: []models.CareInstruction{week1, week3, week4},
	}
	svc := NewInboxService(&fakeInboxBroadcastRepo{}, &fakeInboxAnnouncementRepo{}, &fakeInboxRipenessNoticeRepo{}, &fakeInboxCareGuideService{forCustomer: []models.PlotCareGuide{guide}})

	items, err := svc.GetInboxForCustomer(context.Background(), uuid.New())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(items) != 2 {
		t.Fatalf("got %d items, want 2 (week 4 has not begun yet)", len(items))
	}

	first := items[0]
	if first.Kind != models.InboxItemCare || first.Subject != "Woche 3: Gießen" || first.Body != "Zweimal pro Woche" {
		t.Errorf("first item = %+v, want this week's care instruction first", first)
	}
	if first.FieldName != "Feld Nord" || first.CropName != "Zucchini" || first.FarmName != "" {
		t.Errorf("first item = %+v, want field and crop carried through and no farm", first)
	}
	if want := startAt.Add(14 * 24 * time.Hour); !first.CreatedAt.Equal(want) {
		t.Errorf("first item created_at = %v, want the start of week 3 (%v)", first.CreatedAt, want)
	}
	if items[1].Subject != "Woche 1: Aussäen" || !items[1].CreatedAt.Equal(startAt) {
		t.Errorf("second item = %+v, want week 1 dated at the rental start", items[1])
	}
}

func TestGetInboxForCustomer_SharedInstructionGetsDistinctIDsPerRental(t *testing.T) {
	instruction := models.CareInstruction{ID: uuid.New(), Week: 1, Title: "Aussäen"}
	guide := func() models.PlotCareGuide {
		return models.PlotCareGuide{RentalID: uuid.New(), CurrentWeek: 1, TotalWeeks: 13, Instructions: []models.CareInstruction{instruction}}
	}
	careGuides := &fakeInboxCareGuideService{forCustomer: []models.PlotCareGuide{guide(), guide()}}
	svc := NewInboxService(&fakeInboxBroadcastRepo{}, &fakeInboxAnnouncementRepo{}, &fakeInboxRipenessNoticeRepo{}, careGuides)

	items, err := svc.GetInboxForCustomer(context.Background(), uuid.New())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(items) != 2 {
		t.Fatalf("got %d items, want one per rented plot", len(items))
	}
	if items[0].ID == items[1].ID {
		t.Errorf("both plots' items share id %v, want them distinct", items[0].ID)
	}

	again, err := svc.GetInboxForCustomer(context.Background(), uuid.New())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	ids := map[uuid.UUID]bool{items[0].ID: true, items[1].ID: true}
	if !ids[again[0].ID] || !ids[again[1].ID] {
		t.Errorf("ids changed between reads, want them stable")
	}
}
