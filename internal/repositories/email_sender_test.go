package repositories

import (
	"mime"
	"net/mail"
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
		"Subject: Ernte\r\n",
		"MIME-Version: 1.0\r\n",
		"Content-Type: text/plain; charset=utf-8\r\n\r\n",
		"Samstag um 9",
	} {
		if !strings.Contains(captured.message, want) {
			t.Errorf("message is missing %q\ngot:\n%s", want, captured.message)
		}
	}

	// The address headers are asserted by parsing rather than by matching text:
	// how a name is quoted or encoded is the mail package's business, but what
	// a client reads back out of the header is this package's contract.
	msg, err := mail.ReadMessage(strings.NewReader(captured.message))
	if err != nil {
		t.Fatalf("message does not parse: %v", err)
	}
	for _, tc := range []struct{ header, name, address string }{
		{"From", "BaaS", "noreply@example.com"},
		{"To", "Anna Bauer", "anna@example.com"},
	} {
		addresses, err := msg.Header.AddressList(tc.header)
		if err != nil {
			t.Errorf("parsing %s: %v", tc.header, err)
			continue
		}
		if len(addresses) != 1 {
			t.Errorf("%s lists %d addresses, want 1: %v", tc.header, len(addresses), addresses)
			continue
		}
		if addresses[0].Name != tc.name || addresses[0].Address != tc.address {
			t.Errorf("%s = %v, want %q <%s>", tc.header, addresses[0], tc.name, tc.address)
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
	disposition, params, err := mime.ParseMediaType(headerLine(msg, "Content-Disposition: "))
	if err != nil {
		t.Errorf("parsing the content disposition: %v", err)
	} else if disposition != "attachment" || params["filename"] != "bericht.txt" {
		t.Errorf("content disposition = %q %v, want an attachment named bericht.txt", disposition, params)
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

// headerBlock returns everything before the blank line that ends the headers.
func headerBlock(message string) string {
	block, _, found := strings.Cut(message, "\r\n\r\n")
	if !found {
		return message
	}
	return block
}

func TestSendMail_LineBreaksInTheDisplayNameCannotForgeHeaders(t *testing.T) {
	// first_name and last_name are only trimmed at registration, so a display
	// name is attacker-controlled and must never be able to end its header.
	captured := stubSMTPSend(t)
	sender := NewSMTPEmailSender(testSMTPConfig())

	evil := "Eve\r\nBcc: attacker@evil.example\r\nX-Injected: yes"
	if err := sender.SendMail("eve@example.com", evil, "Ernte", "Hallo", false, nil); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	block := headerBlock(captured.message)
	for _, forged := range []string{"Bcc:", "X-Injected:"} {
		for _, line := range strings.Split(block, "\r\n") {
			if strings.HasPrefix(line, forged) {
				t.Errorf("forged %s header in:\n%s", forged, block)
			}
		}
	}

	msg, err := mail.ReadMessage(strings.NewReader(captured.message))
	if err != nil {
		t.Fatalf("message does not parse: %v", err)
	}
	if got := len(msg.Header); got != 5 {
		t.Errorf("message has %d headers, want the 5 it writes: %v", got, msg.Header)
	}
}

func TestSendMail_LineBreaksInTheSubjectCannotForgeHeaders(t *testing.T) {
	// The subject reaches here straight from the broadcast request body.
	captured := stubSMTPSend(t)
	sender := NewSMTPEmailSender(testSMTPConfig())

	if err := sender.SendMail("anna@example.com", "Anna", "Ernte\r\nX-Injected: yes", "Hallo", false, nil); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	for _, line := range strings.Split(headerBlock(captured.message), "\r\n") {
		if strings.HasPrefix(line, "X-Injected:") {
			t.Errorf("forged header in:\n%s", headerBlock(captured.message))
		}
	}
}

func TestSendMail_AddressSyntaxInANameStaysOneRecipient(t *testing.T) {
	// "Müller, Hans" is an ordinary German name and pure ASCII apart from the
	// umlaut; unquoted, the comma would split the header into two addresses.
	captured := stubSMTPSend(t)
	sender := NewSMTPEmailSender(testSMTPConfig())

	const name = `Müller, Hans "der Bauer"`
	if err := sender.SendMail("hans@example.com", name, "Ernte", "Hallo", false, nil); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	msg, err := mail.ReadMessage(strings.NewReader(captured.message))
	if err != nil {
		t.Fatalf("message does not parse: %v", err)
	}
	addresses, err := msg.Header.AddressList("To")
	if err != nil {
		t.Fatalf("parsing To: %v", err)
	}
	if len(addresses) != 1 {
		t.Fatalf("To lists %d addresses, want 1: %v", len(addresses), addresses)
	}
	if addresses[0].Name != name {
		t.Errorf("name = %q, want %q", addresses[0].Name, name)
	}
	if addresses[0].Address != "hans@example.com" {
		t.Errorf("address = %q, want %q", addresses[0].Address, "hans@example.com")
	}
}

func TestSendMail_AttachmentFilenameIsQuoted(t *testing.T) {
	captured := stubSMTPSend(t)
	sender := NewSMTPEmailSender(testSMTPConfig())

	const name = `ernte";x=y.txt`
	if err := sender.SendMail("anna@example.com", "Anna", "Ernte", "Hallo", false, map[string][]byte{name: []byte("hi")}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	line := headerLine(captured.message, "Content-Disposition: ")
	if line == "" {
		t.Fatalf("no Content-Disposition header in:\n%s", captured.message)
	}
	disposition, params, err := mime.ParseMediaType(line)
	if err != nil {
		t.Fatalf("parsing %q: %v", line, err)
	}
	if disposition != "attachment" {
		t.Errorf("disposition = %q, want %q", disposition, "attachment")
	}
	if params["filename"] != name {
		t.Errorf("filename = %q, want %q", params["filename"], name)
	}
	if _, injected := params["x"]; injected {
		t.Errorf("filename smuggled in a second parameter: %v", params)
	}
}
