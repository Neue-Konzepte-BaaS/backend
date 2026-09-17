package services

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"html/template"
	"io/fs"
	"log/slog"
	"sync"

	"github.com/Neue-Konzepte-BaaS/backend/internal/models"
	"github.com/google/uuid"
)

// NotificationService is the outbound notification provider. Email is the only
// channel today; a second one (push, SMS) plugs in behind this interface
// without any caller having to change.
type NotificationService interface {
	SendMail(email string, displayName string, subject string, message string, isHTML bool, attachments map[string][]byte) error
	SendMailFromTemplate(email string, displayName string, subject string, templateName string, attachments map[string][]byte, templateData any) error
	// SendMailFromTemplateToMany renders the template once per recipient — so a
	// template can greet each by name — and continues past individual
	// failures. It returns how many were delivered and, if any failed, a joined
	// error describing them.
	SendMailFromTemplateToMany(recipients []models.Recipient, subject string, templateName string, data any) (int, error)
	// NotifyAllUsers stores the broadcast and delivers it to every farmer and
	// customer on the platform. It is persisted before recipients are resolved,
	// so a database failure surfaces to the caller before any mail goes out,
	// and the notice remains readable afterwards regardless of delivery
	// outcome. Delivery itself happens in the background: the returned count is
	// how many people were queued, not how many were reached.
	NotifyAllUsers(ctx context.Context, subject, body string) (int, error)
	// NotifyFarmerCustomers delivers a message to the customers currently
	// renting one of the farmer's plots. It behaves like NotifyAllUsers:
	// recipients are resolved before returning, delivery happens afterwards,
	// and the count is how many were queued.
	NotifyFarmerCustomers(ctx context.Context, farmer uuid.UUID, farmName, subject, body string) (int, error)
}

// RecipientData is what a notification template is executed against: the
// payload the caller supplied, plus the person the copy is going to. Templates
// reach them as {{ .Recipient.DisplayName }} and {{ .Data.Subject }}.
type RecipientData struct {
	Recipient models.Recipient
	Data      any
}

// broadcastData is the payload for the platform-wide broadcast template.
type broadcastData struct {
	Subject string
	Body    string
}

// announcementData is the payload for a farmer's announcement. It carries the
// farm name because a customer rents from several farmers and the mail is only
// meaningful if it says which one is writing.
type announcementData struct {
	FarmName string
	Subject  string
	Body     string
}

const (
	broadcastTemplate    = "broadcast"
	announcementTemplate = "announcement"
)

type templateEntry struct {
	once sync.Once
	tmpl *template.Template
	err  error
}

type notificationService struct {
	emailSender   EmailSender
	accountRepo   AccountRepository
	broadcastRepo BroadcastNotificationRepository
	templateFS    fs.FS
	dispatcher    *Dispatcher
	templates     sync.Map
}

func NewNotificationService(emailSender EmailSender, accountRepo AccountRepository, broadcastRepo BroadcastNotificationRepository, templateFS fs.FS, dispatcher *Dispatcher) NotificationService {
	return &notificationService{
		emailSender:   emailSender,
		accountRepo:   accountRepo,
		broadcastRepo: broadcastRepo,
		templateFS:    templateFS,
		dispatcher:    dispatcher,
	}
}

// getTemplate parses a template on first use and caches it. The sync.Once per
// entry means concurrent requests for the same template parse it once, and a
// parse failure is cached too rather than retried on every send.
func (s *notificationService) getTemplate(templateName string) (*template.Template, error) {
	actual, _ := s.templates.LoadOrStore(templateName, &templateEntry{})
	entry := actual.(*templateEntry)

	entry.once.Do(func() {
		entry.tmpl, entry.err = template.ParseFS(s.templateFS, templateName+".html")
	})

	return entry.tmpl, entry.err
}

func (s *notificationService) SendMail(email string, displayName string, subject string, message string, isHTML bool, attachments map[string][]byte) error {
	return s.emailSender.SendMail(email, displayName, subject, message, isHTML, attachments)
}

func (s *notificationService) SendMailFromTemplate(email string, displayName string, subject string, templateName string, attachments map[string][]byte, templateData any) error {
	tmpl, err := s.getTemplate(templateName)
	if err != nil {
		return fmt.Errorf("parsing template %q: %w", templateName, err)
	}

	var buffer bytes.Buffer
	if err := tmpl.Execute(&buffer, templateData); err != nil {
		return fmt.Errorf("executing template %q: %w", templateName, err)
	}

	if err := s.emailSender.SendMail(email, displayName, subject, buffer.String(), true, attachments); err != nil {
		return fmt.Errorf("sending email to %q: %w", email, err)
	}
	return nil
}

func (s *notificationService) SendMailFromTemplateToMany(recipients []models.Recipient, subject string, templateName string, data any) (int, error) {
	// Fail before the first send rather than once per recipient: a broken
	// template is a bug in this repo, not a per-recipient condition.
	if _, err := s.getTemplate(templateName); err != nil {
		return 0, fmt.Errorf("parsing template %q: %w", templateName, err)
	}

	var (
		sent int
		errs []error
	)
	for _, recipient := range recipients {
		err := s.SendMailFromTemplate(
			recipient.Email,
			recipient.DisplayName(),
			subject,
			templateName,
			nil,
			RecipientData{Recipient: recipient, Data: data},
		)
		if err != nil {
			errs = append(errs, err)
			continue
		}
		sent++
	}

	return sent, errors.Join(errs...)
}

func (s *notificationService) NotifyAllUsers(ctx context.Context, subject, body string) (int, error) {
	if _, err := s.broadcastRepo.CreateBroadcastNotification(ctx, subject, body); err != nil {
		return 0, fmt.Errorf("storing broadcast notification: %w", err)
	}

	recipients, err := s.accountRepo.GetAllRecipients(ctx)
	if err != nil {
		return 0, fmt.Errorf("loading recipients: %w", err)
	}
	if len(recipients) == 0 {
		return 0, nil
	}

	s.deliverInBackground(recipients, subject, broadcastTemplate, broadcastData{Subject: subject, Body: body}, "broadcast")

	return len(recipients), nil
}

func (s *notificationService) NotifyFarmerCustomers(ctx context.Context, farmer uuid.UUID, farmName, subject, body string) (int, error) {
	recipients, err := s.accountRepo.GetCustomersOfFarmer(ctx, farmer)
	if err != nil {
		return 0, fmt.Errorf("loading customers of farmer %s: %w", farmer, err)
	}
	if len(recipients) == 0 {
		return 0, nil
	}

	s.deliverInBackground(recipients, subject, announcementTemplate,
		announcementData{FarmName: farmName, Subject: subject, Body: body}, "announcement")

	return len(recipients), nil
}

// deliverInBackground hands a fan-out to the dispatcher. The whole fan-out is
// one task, so sends within it are serial and the dispatcher's concurrency
// bounds how many fan-outs run at once — which keeps the load on the relay
// predictable no matter how large a single audience is.
func (s *notificationService) deliverInBackground(recipients []models.Recipient, subject, templateName string, data any, kind string) {
	s.dispatcher.Go(func() {
		sent, err := s.SendMailFromTemplateToMany(recipients, subject, templateName, data)
		if err != nil {
			slog.Error("delivering notifications failed for some recipients",
				"kind", kind, "sent", sent, "total", len(recipients), "error", err)
			return
		}
		slog.Info("notifications delivered", "kind", kind, "sent", sent)
	})
}
