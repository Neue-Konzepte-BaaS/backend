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
	// NotifyAllUsers delivers a message to every farmer and customer on the
	// platform. Recipients are resolved before returning, so a database failure
	// surfaces to the caller, but delivery itself happens in the background:
	// the returned count is how many people were queued, not how many were
	// reached.
	NotifyAllUsers(ctx context.Context, subject, body string) (int, error)
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

const broadcastTemplate = "broadcast"

type templateEntry struct {
	once sync.Once
	tmpl *template.Template
	err  error
}

type notificationService struct {
	emailSender EmailSender
	accountRepo AccountRepository
	templateFS  fs.FS
	dispatcher  *Dispatcher
	templates   sync.Map
}

func NewNotificationService(emailSender EmailSender, accountRepo AccountRepository, templateFS fs.FS, dispatcher *Dispatcher) NotificationService {
	return &notificationService{
		emailSender: emailSender,
		accountRepo: accountRepo,
		templateFS:  templateFS,
		dispatcher:  dispatcher,
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
