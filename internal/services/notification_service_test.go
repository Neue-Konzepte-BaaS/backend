package services

import (
	"context"
	"errors"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/Neue-Konzepte-BaaS/backend/internal/models"
	"github.com/google/uuid"
)

// sentMail is one delivery recorded by fakeEmailSender.
type sentMail struct {
	email       string
	displayName string
	subject     string
	message     string
	isHTML      bool
}

// fakeEmailSender is an in-memory EmailSender. failFor names addresses whose
// delivery should fail, so a test can prove one bad recipient does not stop the
// rest of a fan-out.
type fakeEmailSender struct {
	sent    []sentMail
	failFor map[string]error
}

func (f *fakeEmailSender) SendMail(email, displayName, subject, message string, isHTML bool, _ map[string][]byte) error {
	if err, ok := f.failFor[email]; ok {
		return err
	}
	f.sent = append(f.sent, sentMail{
		email:       email,
		displayName: displayName,
		subject:     subject,
		message:     message,
		isHTML:      isHTML,
	})
	return nil
}

// fakeRecipientRepo is an AccountRepository that only answers recipient
// lookups; the rest of the interface is unreachable from this service.
type fakeRecipientRepo struct {
	recipients        []models.Recipient
	customersOfFarmer map[uuid.UUID][]models.Recipient
	err               error
	calls             int
}

func (f *fakeRecipientRepo) GetAllRecipients(context.Context) ([]models.Recipient, error) {
	f.calls++
	if f.err != nil {
		return nil, f.err
	}
	return f.recipients, nil
}

// customersOfFarmer is keyed by farmer id, so one fake can answer both the
// platform-wide lookup and the per-farmer one.
func (f *fakeRecipientRepo) GetCustomersOfFarmer(_ context.Context, farmer uuid.UUID) ([]models.Recipient, error) {
	f.calls++
	if f.err != nil {
		return nil, f.err
	}
	return f.customersOfFarmer[farmer], nil
}

func (f *fakeRecipientRepo) GetAccountByEmail(context.Context, string) (models.Account, error) {
	panic("notification service does not look accounts up by email")
}

func (f *fakeRecipientRepo) GetAccountByID(context.Context, uuid.UUID) (models.Account, error) {
	panic("notification service does not look accounts up by id")
}

func (f *fakeRecipientRepo) CreateAdmin(context.Context, models.Account) (models.Account, error) {
	panic("notification service does not create accounts")
}

func (f *fakeRecipientRepo) CreateFarmer(context.Context, models.Account, string, int32, string, string) (models.Account, error) {
	panic("notification service does not create accounts")
}

func (f *fakeRecipientRepo) CreateCustomer(context.Context, models.Account, int32) (models.Account, error) {
	panic("notification service does not create accounts")
}

// fakeBroadcastRepo records what was stored and can fail on demand.
type fakeBroadcastRepo struct {
	stored    []models.BroadcastNotification
	createErr error
}

func (f *fakeBroadcastRepo) CreateBroadcastNotification(_ context.Context, subject, body string) (models.BroadcastNotification, error) {
	if f.createErr != nil {
		return models.BroadcastNotification{}, f.createErr
	}
	notification := models.BroadcastNotification{ID: uuid.New(), Subject: subject, Body: body}
	f.stored = append(f.stored, notification)
	return notification, nil
}

func (f *fakeBroadcastRepo) GetAllBroadcastNotifications(context.Context) ([]models.BroadcastNotification, error) {
	return f.stored, nil
}

func testTemplates() fstest.MapFS {
	return fstest.MapFS{
		"broadcast.html": &fstest.MapFile{
			Data: []byte("<p>Hallo {{ .Recipient.DisplayName }}</p><h1>{{ .Data.Subject }}</h1><p>{{ .Data.Body }}</p>"),
		},
	}
}

func newTestNotificationService(sender EmailSender, repo AccountRepository) NotificationService {
	return NewNotificationService(sender, repo, &fakeBroadcastRepo{}, testTemplates(), NewDispatcher(1))
}

func recipient(email, first, last string) models.Recipient {
	return models.Recipient{AccountID: uuid.New(), Email: email, FirstName: first, LastName: last}
}

func TestSendMailFromTemplate_RendersAndSendsAsHTML(t *testing.T) {
	sender := &fakeEmailSender{}
	svc := newTestNotificationService(sender, &fakeRecipientRepo{})

	data := RecipientData{
		Recipient: recipient("anna@example.com", "Anna", "Bauer"),
		Data:      broadcastData{Subject: "Ernte", Body: "Samstag"},
	}
	if err := svc.SendMailFromTemplate("anna@example.com", "Anna Bauer", "Ernte", "broadcast", nil, data); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(sender.sent) != 1 {
		t.Fatalf("sent %d mails, want 1", len(sender.sent))
	}
	got := sender.sent[0]
	if !got.isHTML {
		t.Error("a templated mail must be sent as HTML")
	}
	if !strings.Contains(got.message, "Hallo Anna Bauer") {
		t.Errorf("message = %q, want it to greet the recipient by name", got.message)
	}
	if !strings.Contains(got.message, "Samstag") {
		t.Errorf("message = %q, want it to contain the body", got.message)
	}
}

func TestSendMailFromTemplate_UnknownTemplateIsAnError(t *testing.T) {
	sender := &fakeEmailSender{}
	svc := newTestNotificationService(sender, &fakeRecipientRepo{})

	err := svc.SendMailFromTemplate("anna@example.com", "Anna", "Ernte", "does-not-exist", nil, nil)
	if err == nil {
		t.Fatal("expected an error for a missing template")
	}
	if len(sender.sent) != 0 {
		t.Errorf("sent %d mails, want 0 when the template is missing", len(sender.sent))
	}
}

func TestSendMailFromTemplateToMany_SendsOnePersonalisedMailPerRecipient(t *testing.T) {
	sender := &fakeEmailSender{}
	svc := newTestNotificationService(sender, &fakeRecipientRepo{})

	recipients := []models.Recipient{
		recipient("anna@example.com", "Anna", "Bauer"),
		recipient("ben@example.com", "Ben", "Klein"),
	}

	sent, err := svc.SendMailFromTemplateToMany(recipients, "Ernte", "broadcast", broadcastData{Subject: "Ernte", Body: "Samstag"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if sent != 2 {
		t.Errorf("sent = %d, want 2", sent)
	}
	if len(sender.sent) != 2 {
		t.Fatalf("recorded %d mails, want 2", len(sender.sent))
	}
	if !strings.Contains(sender.sent[0].message, "Hallo Anna Bauer") {
		t.Errorf("first mail = %q, want it addressed to Anna Bauer", sender.sent[0].message)
	}
	if !strings.Contains(sender.sent[1].message, "Hallo Ben Klein") {
		t.Errorf("second mail = %q, want it addressed to Ben Klein", sender.sent[1].message)
	}
}

func TestSendMailFromTemplateToMany_OneFailureDoesNotStopTheRest(t *testing.T) {
	boom := errors.New("relay refused")
	sender := &fakeEmailSender{failFor: map[string]error{"ben@example.com": boom}}
	svc := newTestNotificationService(sender, &fakeRecipientRepo{})

	recipients := []models.Recipient{
		recipient("anna@example.com", "Anna", "Bauer"),
		recipient("ben@example.com", "Ben", "Klein"),
		recipient("cara@example.com", "Cara", "Lang"),
	}

	sent, err := svc.SendMailFromTemplateToMany(recipients, "Ernte", "broadcast", broadcastData{Subject: "Ernte"})
	if err == nil {
		t.Fatal("expected the failed recipient to be reported")
	}
	if !errors.Is(err, boom) {
		t.Errorf("error = %v, want it to wrap the sender failure", err)
	}
	if sent != 2 {
		t.Errorf("sent = %d, want 2 of 3 delivered", sent)
	}
	if len(sender.sent) != 2 {
		t.Errorf("recorded %d mails, want the two that succeeded", len(sender.sent))
	}
}

func TestSendMailFromTemplateToMany_NoRecipientsIsNotAnError(t *testing.T) {
	sender := &fakeEmailSender{}
	svc := newTestNotificationService(sender, &fakeRecipientRepo{})

	sent, err := svc.SendMailFromTemplateToMany(nil, "Ernte", "broadcast", broadcastData{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if sent != 0 {
		t.Errorf("sent = %d, want 0", sent)
	}
}

func TestNotifyAllUsers_QueuesEveryRecipient(t *testing.T) {
	sender := &fakeEmailSender{}
	repo := &fakeRecipientRepo{recipients: []models.Recipient{
		recipient("anna@example.com", "Anna", "Bauer"),
		recipient("ben@example.com", "Ben", "Klein"),
	}}
	dispatcher := NewDispatcher(1)
	broadcastRepo := &fakeBroadcastRepo{}
	svc := NewNotificationService(sender, repo, broadcastRepo, testTemplates(), dispatcher)

	queued, err := svc.NotifyAllUsers(context.Background(), "Wartung", "Sonntag offline")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if queued != 2 {
		t.Errorf("queued = %d, want 2", queued)
	}
	if repo.calls != 1 {
		t.Errorf("recipient lookups = %d, want exactly 1", repo.calls)
	}
	if len(broadcastRepo.stored) != 1 {
		t.Errorf("stored %d broadcasts, want 1", len(broadcastRepo.stored))
	}

	// Delivery is asynchronous, so wait for the dispatcher before asserting.
	if err := dispatcher.Wait(context.Background()); err != nil {
		t.Fatalf("waiting for delivery: %v", err)
	}
	if len(sender.sent) != 2 {
		t.Fatalf("delivered %d mails, want 2", len(sender.sent))
	}
	if sender.sent[0].subject != "Wartung" {
		t.Errorf("subject = %q, want %q", sender.sent[0].subject, "Wartung")
	}
}

func TestNotifyAllUsers_RepositoryFailureIsReturned(t *testing.T) {
	boom := errors.New("db exploded")
	repo := &fakeRecipientRepo{err: boom}
	svc := newTestNotificationService(&fakeEmailSender{}, repo)

	if _, err := svc.NotifyAllUsers(context.Background(), "Wartung", "Sonntag offline"); !errors.Is(err, boom) {
		t.Errorf("error = %v, want it to wrap the repository failure", err)
	}
}

func TestNotifyAllUsers_StorageFailurePreventsSending(t *testing.T) {
	boom := errors.New("db exploded")
	sender := &fakeEmailSender{}
	repo := &fakeRecipientRepo{recipients: []models.Recipient{recipient("anna@example.com", "Anna", "Bauer")}}
	broadcastRepo := &fakeBroadcastRepo{createErr: boom}
	svc := NewNotificationService(sender, repo, broadcastRepo, testTemplates(), NewDispatcher(1))

	if _, err := svc.NotifyAllUsers(context.Background(), "Wartung", "Sonntag offline"); !errors.Is(err, boom) {
		t.Errorf("error = %v, want it to wrap the storage failure", err)
	}
	if repo.calls != 0 {
		t.Errorf("recipient lookups = %d, want 0 when storage failed first", repo.calls)
	}
	if len(sender.sent) != 0 {
		t.Errorf("sent %d mails, want none when storage failed", len(sender.sent))
	}
}

func TestNotifyAllUsers_NoAccountsSendsNothing(t *testing.T) {
	sender := &fakeEmailSender{}
	svc := newTestNotificationService(sender, &fakeRecipientRepo{})

	queued, err := svc.NotifyAllUsers(context.Background(), "Wartung", "Sonntag offline")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if queued != 0 {
		t.Errorf("queued = %d, want 0", queued)
	}
	if len(sender.sent) != 0 {
		t.Errorf("sent %d mails, want none", len(sender.sent))
	}
}

func TestNotifyFarmerCustomers_QueuesTheFarmersOwnCustomers(t *testing.T) {
	farmer := uuid.New()
	sender := &fakeEmailSender{}
	repo := &fakeRecipientRepo{customersOfFarmer: map[uuid.UUID][]models.Recipient{
		farmer: {
			recipient("anna@example.com", "Anna", "Bauer"),
			recipient("ben@example.com", "Ben", "Klein"),
		},
	}}
	dispatcher := NewDispatcher(1)
	svc := NewNotificationService(sender, repo, &fakeBroadcastRepo{}, announcementTestTemplates(), dispatcher)

	queued, err := svc.NotifyFarmerCustomers(context.Background(), farmer, "Hof Grünwald", "Ernte", "Samstag")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if queued != 2 {
		t.Errorf("queued = %d, want 2", queued)
	}

	if err := dispatcher.Wait(context.Background()); err != nil {
		t.Fatalf("waiting for delivery: %v", err)
	}
	if len(sender.sent) != 2 {
		t.Fatalf("delivered %d mails, want 2", len(sender.sent))
	}
	// The farm name is the whole reason the announcement template differs from
	// the broadcast one: a customer rents from several farmers.
	if !strings.Contains(sender.sent[0].message, "Hof Grünwald") {
		t.Errorf("message = %q, want it to name the farm", sender.sent[0].message)
	}
	if !strings.Contains(sender.sent[0].message, "Hallo Anna Bauer") {
		t.Errorf("message = %q, want it to greet the recipient", sender.sent[0].message)
	}
}

func TestNotifyFarmerCustomers_NoCustomersSendsNothing(t *testing.T) {
	sender := &fakeEmailSender{}
	repo := &fakeRecipientRepo{customersOfFarmer: map[uuid.UUID][]models.Recipient{}}
	svc := NewNotificationService(sender, repo, &fakeBroadcastRepo{}, announcementTestTemplates(), NewDispatcher(1))

	queued, err := svc.NotifyFarmerCustomers(context.Background(), uuid.New(), "Hof", "Ernte", "Samstag")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if queued != 0 {
		t.Errorf("queued = %d, want 0", queued)
	}
	if len(sender.sent) != 0 {
		t.Errorf("sent %d mails, want none", len(sender.sent))
	}
}

func TestNotifyFarmerCustomers_RepositoryFailureIsReturned(t *testing.T) {
	boom := errors.New("db exploded")
	svc := NewNotificationService(&fakeEmailSender{}, &fakeRecipientRepo{err: boom}, &fakeBroadcastRepo{}, announcementTestTemplates(), NewDispatcher(1))

	if _, err := svc.NotifyFarmerCustomers(context.Background(), uuid.New(), "Hof", "Ernte", "Samstag"); !errors.Is(err, boom) {
		t.Errorf("error = %v, want it to wrap the repository failure", err)
	}
}

// announcementTestTemplates adds the announcement body to the broadcast one so
// a single fake filesystem serves both fan-outs.
func announcementTestTemplates() fstest.MapFS {
	templates := testTemplates()
	templates["announcement.html"] = &fstest.MapFile{
		Data: []byte("<p>Hallo {{ .Recipient.DisplayName }}</p><p>{{ .Data.FarmName }}</p><h1>{{ .Data.Subject }}</h1><p>{{ .Data.Body }}</p>"),
	}
	return templates
}
