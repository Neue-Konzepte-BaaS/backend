package services

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/Neue-Konzepte-BaaS/backend/internal/models"
	"github.com/google/uuid"
)

// fakeAnnouncementRepo records what was stored and can fail on demand.
type fakeAnnouncementRepo struct {
	created      []models.AnnouncementWithFarm
	byFarmer     []models.AnnouncementWithFarm
	forCustomer  []models.AnnouncementWithFarm
	createErr    error
	listErr      error
	farmName     string
	createdCalls int
}

func (f *fakeAnnouncementRepo) CreateAnnouncement(_ context.Context, farmer uuid.UUID, subject, body string) (models.AnnouncementWithFarm, error) {
	f.createdCalls++
	if f.createErr != nil {
		return models.AnnouncementWithFarm{}, f.createErr
	}
	announcement := models.AnnouncementWithFarm{
		Announcement: models.Announcement{
			ID:        uuid.New(),
			Farmer:    farmer,
			Subject:   subject,
			Body:      body,
			CreatedAt: time.Now(),
		},
		FarmName: f.farmName,
	}
	f.created = append(f.created, announcement)
	return announcement, nil
}

func (f *fakeAnnouncementRepo) GetAnnouncementsByFarmer(context.Context, uuid.UUID) ([]models.AnnouncementWithFarm, error) {
	if f.listErr != nil {
		return nil, f.listErr
	}
	return f.byFarmer, nil
}

func (f *fakeAnnouncementRepo) GetAnnouncementsForCustomer(context.Context, uuid.UUID) ([]models.AnnouncementWithFarm, error) {
	if f.listErr != nil {
		return nil, f.listErr
	}
	return f.forCustomer, nil
}

// fakeNotifier stands in for the notification provider. Only the fan-out the
// board uses is reachable from this service.
type fakeNotifier struct {
	NotificationService

	queued     int
	err        error
	calls      int
	gotFarmer  uuid.UUID
	gotFarm    string
	gotSubject string
	gotBody    string
}

func (f *fakeNotifier) NotifyFarmerCustomers(_ context.Context, farmer uuid.UUID, farmName, subject, body string) (int, error) {
	f.calls++
	f.gotFarmer, f.gotFarm, f.gotSubject, f.gotBody = farmer, farmName, subject, body
	if f.err != nil {
		return 0, f.err
	}
	return f.queued, nil
}

func TestCreateAnnouncement_StoresThenMailsTheFarmersCustomers(t *testing.T) {
	farmer := uuid.New()
	repo := &fakeAnnouncementRepo{farmName: "Hof Grünwald"}
	notifier := &fakeNotifier{queued: 3}
	svc := NewAnnouncementService(repo, notifier)

	announcement, recipients, err := svc.CreateAnnouncement(context.Background(), farmer, "Ernte", "Samstag um 9")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(repo.created) != 1 {
		t.Fatalf("stored %d announcements, want 1", len(repo.created))
	}
	if announcement.Subject != "Ernte" || announcement.Body != "Samstag um 9" {
		t.Errorf("announcement = %+v, want the posted subject and body", announcement)
	}
	if announcement.FarmName != "Hof Grünwald" {
		t.Errorf("farm name = %q, want it resolved by the repository", announcement.FarmName)
	}
	if recipients != 3 {
		t.Errorf("recipients = %d, want 3", recipients)
	}

	// The fan-out must be told which farm is writing, or the mail cannot say.
	if notifier.calls != 1 {
		t.Fatalf("notifier called %d times, want 1", notifier.calls)
	}
	if notifier.gotFarmer != farmer || notifier.gotFarm != "Hof Grünwald" {
		t.Errorf("notified farmer %v / farm %q, want %v / %q", notifier.gotFarmer, notifier.gotFarm, farmer, "Hof Grünwald")
	}
	if notifier.gotSubject != "Ernte" || notifier.gotBody != "Samstag um 9" {
		t.Errorf("notified with %q / %q, want the posted subject and body", notifier.gotSubject, notifier.gotBody)
	}
}

func TestCreateAnnouncement_StoreFailureSendsNothing(t *testing.T) {
	boom := errors.New("db exploded")
	repo := &fakeAnnouncementRepo{createErr: boom}
	notifier := &fakeNotifier{}
	svc := NewAnnouncementService(repo, notifier)

	if _, _, err := svc.CreateAnnouncement(context.Background(), uuid.New(), "Ernte", "Samstag"); !errors.Is(err, boom) {
		t.Errorf("error = %v, want it to wrap the repository failure", err)
	}
	if notifier.calls != 0 {
		t.Errorf("notifier called %d times, want 0 when nothing was stored", notifier.calls)
	}
}

func TestCreateAnnouncement_MailFailureStillKeepsTheAnnouncement(t *testing.T) {
	// The board is what a customer who misses the mail reads instead, so a
	// delivery failure must not undo the post or report it as failed.
	repo := &fakeAnnouncementRepo{farmName: "Hof Grünwald"}
	notifier := &fakeNotifier{err: errors.New("relay refused")}
	svc := NewAnnouncementService(repo, notifier)

	announcement, recipients, err := svc.CreateAnnouncement(context.Background(), uuid.New(), "Ernte", "Samstag")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if announcement.Subject != "Ernte" {
		t.Errorf("announcement = %+v, want it returned despite the mail failure", announcement)
	}
	if recipients != 0 {
		t.Errorf("recipients = %d, want 0 when nothing could be queued", recipients)
	}
	if len(repo.created) != 1 {
		t.Errorf("stored %d announcements, want the post to survive", len(repo.created))
	}
}

func TestGetAnnouncements_FarmerAndCustomerReadDifferentBoards(t *testing.T) {
	mine := models.AnnouncementWithFarm{
		Announcement: models.Announcement{Subject: "meine"},
		FarmName:     "Hof Grünwald",
	}
	theirs := models.AnnouncementWithFarm{
		Announcement: models.Announcement{Subject: "vom Hof"},
		FarmName:     "Hof Klein",
	}
	repo := &fakeAnnouncementRepo{byFarmer: []models.AnnouncementWithFarm{mine}, forCustomer: []models.AnnouncementWithFarm{theirs}}
	svc := NewAnnouncementService(repo, &fakeNotifier{})

	farmerBoard, err := svc.GetAnnouncementsForFarmer(context.Background(), uuid.New())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(farmerBoard) != 1 || farmerBoard[0].Subject != "meine" {
		t.Errorf("farmer board = %+v, want only the farmer's own notices", farmerBoard)
	}

	customerBoard, err := svc.GetAnnouncementsForCustomer(context.Background(), uuid.New())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(customerBoard) != 1 || customerBoard[0].Subject != "vom Hof" {
		t.Errorf("customer board = %+v, want the notices of farmers rented from", customerBoard)
	}
	if customerBoard[0].FarmName != "Hof Klein" {
		t.Errorf("farm name = %q, want each notice to say which farm it came from", customerBoard[0].FarmName)
	}
}

func TestGetAnnouncements_RepositoryFailureIsReturned(t *testing.T) {
	boom := errors.New("db exploded")
	svc := NewAnnouncementService(&fakeAnnouncementRepo{listErr: boom}, &fakeNotifier{})

	if _, err := svc.GetAnnouncementsForFarmer(context.Background(), uuid.New()); !errors.Is(err, boom) {
		t.Errorf("farmer board error = %v, want it to wrap the repository failure", err)
	}
	if _, err := svc.GetAnnouncementsForCustomer(context.Background(), uuid.New()); !errors.Is(err, boom) {
		t.Errorf("customer board error = %v, want it to wrap the repository failure", err)
	}
}
