package repositories

import (
	"mime"
	"net/smtp"
	"strings"
	"testing"
)

// capturedSend records what the sender handed to smtp.SendMail.
type capturedSend struct {
	addr      string
	auth      smtp.Auth
	from      string
	receivers []string
	message   string
	calls     int
}

// stubSMTPSend swaps the package-level smtpSend for the duration of a test and
// returns the recorder it writes into.
func stubSMTPSend(t *testing.T) *capturedSend {
	t.Helper()

	captured := &capturedSend{}
	original := smtpSend
	smtpSend = func(addr string, auth smtp.Auth, from string, to []string, msg []byte) error {
		captured.calls++
		captured.addr = addr
		captured.auth = auth
		captured.from = from
		captured.receivers = to
		captured.message = string(msg)
		return nil
	}
	t.Cleanup(func() { smtpSend = original })

	return captured
}

func testSMTPConfig() SMTPConfig {
	return SMTPConfig{
		Hostname:    "smtp.example.com",
		Port:        587,
		Username:    "mailer",
		Password:    "secret",
		SenderName:  "BaaS",
		SenderEmail: "noreply@example.com",
	}
}

func TestSendMail_PlainTextHeadersAndBody(t *testing.T) {
	captured := stubSMTPSend(t)
	sender := NewSMTPEmailSender(testSMTPConfig())

	if err := sender.SendMail("anna@example.com", "Anna Bauer", "Ernte", "Samstag um 9", false, nil); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if captured.calls != 1 {
		t.Fatalf("smtpSend called %d times, want 1", captured.calls)
	}
	if captured.addr != "smtp.example.com:587" {
		t.Errorf("addr = %q, want %q", captured.addr, "smtp.example.com:587")
	}
	if captured.from != "noreply@example.com" {
		t.Errorf("from = %q, want %q", captured.from, "noreply@example.com")
	}
	if len(captured.receivers) != 1 || captured.receivers[0] != "anna@example.com" {
		t.Errorf("receivers = %v, want [anna@example.com]", captured.receivers)
	}

	for _, want := range []string{
		"From: BaaS <noreply@example.com>\r\n",
		"Subject: Ernte\r\n",
		"To: Anna Bauer <anna@example.com>\r\n",
		"MIME-Version: 1.0\r\n",
		"Content-Type: text/plain; charset=utf-8\r\n\r\n",
		"Samstag um 9",
	} {
		if !strings.Contains(captured.message, want) {
			t.Errorf("message is missing %q\ngot:\n%s", want, captured.message)
		}
	}
}

func TestSendMail_HTMLUsesHTMLContentType(t *testing.T) {
	captured := stubSMTPSend(t)
	sender := NewSMTPEmailSender(testSMTPConfig())

	if err := sender.SendMail("anna@example.com", "Anna", "Ernte", "<p>Hallo</p>", true, nil); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(captured.message, "Content-Type: text/html; charset=utf-8\r\n\r\n") {
		t.Errorf("want an html content type with charset utf-8, got:\n%s", captured.message)
	}
}

func TestSendMail_NonASCIIHeadersAreRFC2047Encoded(t *testing.T) {
	captured := stubSMTPSend(t)
	config := testSMTPConfig()
	config.SenderName = "Bauernhof Grünwald"
	sender := NewSMTPEmailSender(config)

	subject := "Frühjahrsaussaat"
	if err := sender.SendMail("anna@example.com", "Anna Müller", subject, "Hallo", false, nil); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// The raw values must not appear unencoded, and the encoded forms must
	// decode back to the originals.
	if strings.Contains(captured.message, subject) {
		t.Errorf("subject %q was sent unencoded", subject)
	}

	decoder := new(mime.WordDecoder)
	for _, tc := range []struct{ prefix, want string }{
		{"Subject: ", subject},
		{"From: ", "Bauernhof Grünwald"},
		{"To: ", "Anna Müller"},
	} {
		line := headerLine(captured.message, tc.prefix)
		if line == "" {
			t.Fatalf("no %q header in:\n%s", tc.prefix, captured.message)
		}
		encoded := strings.Fields(line)[0]
		got, err := decoder.Decode(encoded)
		if err != nil {
			t.Errorf("decoding %q: %v", encoded, err)
			continue
		}
		if got != tc.want {
			t.Errorf("decoded %s= %q, want %q", tc.prefix, got, tc.want)
		}
	}
}

// headerLine returns the value of the header starting with prefix.
func headerLine(message, prefix string) string {
	for _, line := range strings.Split(message, "\r\n") {
		if strings.HasPrefix(line, prefix) {
			return strings.TrimPrefix(line, prefix)
		}
	}
	return ""
}

func TestSendMail_AttachmentIsBase64AndWrapped(t *testing.T) {
	captured := stubSMTPSend(t)
	sender := NewSMTPEmailSender(testSMTPConfig())

	// Long enough that base64 spans several 76-column lines.
	content := []byte(strings.Repeat("Feldbericht. ", 40))
	attachments := map[string][]byte{"bericht.txt": content}

	if err := sender.SendMail("anna@example.com", "Anna", "Bericht", "siehe Anhang", false, attachments); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	msg := captured.message
	if !strings.Contains(msg, "Content-Type: multipart/mixed; boundary=") {
		t.Fatalf("want a multipart content type, got:\n%s", msg)
	}
	if !strings.Contains(msg, "Content-Transfer-Encoding: base64") {
		t.Error("want a base64 transfer encoding for the attachment")
	}
	if !strings.Contains(msg, `Content-Disposition: attachment; filename="bericht.txt"`) {
		t.Error("want a content disposition naming the attachment")
	}
	if !strings.Contains(msg, "Content-Type: text/plain; charset=utf-8") {
		t.Error("want the message part to keep charset utf-8 inside a multipart body")
	}

	boundary := headerLine(msg, "Content-Type: multipart/mixed; boundary=")
	if boundary == "" {
		t.Fatal("could not read the multipart boundary")
	}
	// Close() writes the terminating boundary exactly once; a second Close
	// would emit it twice and confuse strict clients.
	if got := strings.Count(msg, "--"+boundary+"--"); got != 1 {
		t.Errorf("terminating boundary appears %d times, want 1", got)
	}

	for _, line := range strings.Split(msg, "\r\n") {
		if len(line) > 998 {
			t.Errorf("line exceeds the SMTP limit at %d chars", len(line))
		}
	}
}

func TestSendMail_UnknownExtensionFallsBackToOctetStream(t *testing.T) {
	captured := stubSMTPSend(t)
	sender := NewSMTPEmailSender(testSMTPConfig())

	attachments := map[string][]byte{"daten.unknownext": []byte("payload")}
	if err := sender.SendMail("anna@example.com", "Anna", "Daten", "siehe Anhang", false, attachments); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(captured.message, "Content-Type: application/octet-stream") {
		t.Errorf("want an octet-stream fallback for an unknown extension, got:\n%s", captured.message)
	}
	if strings.Contains(captured.message, "Content-Type: \r\n") {
		t.Error("an empty content type is not a legal header value")
	}
}

func TestNewSMTPEmailSender_NoUsernameMeansNoAuth(t *testing.T) {
	captured := stubSMTPSend(t)
	config := testSMTPConfig()
	config.Username = ""
	sender := NewSMTPEmailSender(config)

	if err := sender.SendMail("anna@example.com", "Anna", "Ernte", "Hallo", false, nil); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// A relay that wants no credentials rejects a PLAIN handshake outright.
	if captured.auth != nil {
		t.Error("auth must be nil when no username is configured")
	}
}

func TestNewSMTPEmailSender_UsernameMeansAuth(t *testing.T) {
	captured := stubSMTPSend(t)
	sender := NewSMTPEmailSender(testSMTPConfig())

	if err := sender.SendMail("anna@example.com", "Anna", "Ernte", "Hallo", false, nil); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if captured.auth == nil {
		t.Error("auth must be set when a username is configured")
	}
}

func TestConsoleEmailSender_NeverFails(t *testing.T) {
	sender := NewConsoleEmailSender()

	if err := sender.SendMail("anna@example.com", "Anna", "Ernte", "Hallo", false, nil); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}
