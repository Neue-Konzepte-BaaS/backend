package repositories

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"log/slog"
	"mime"
	"mime/multipart"
	"net/mail"
	"net/smtp"
	"net/textproto"
	"path/filepath"
	"strconv"
	"strings"
	"unicode"

	"github.com/Neue-Konzepte-BaaS/backend/internal/services"
)

// base64LineLength is the maximum line length RFC 2045 allows for a base64
// transfer encoding.
const base64LineLength = 76

// SMTPConfig holds the configuration for an SMTP email sender.
type SMTPConfig struct {
	Hostname    string
	Port        int
	Username    string
	Password    string
	SenderName  string
	SenderEmail string
}

type smtpEmailSender struct {
	config SMTPConfig
	auth   smtp.Auth
}

// NewSMTPEmailSender creates a new EmailSender backed by SMTP.
func NewSMTPEmailSender(config SMTPConfig) services.EmailSender {
	// A relay that wants no credentials rejects the AUTH command outright, so
	// an empty username means no auth rather than empty auth.
	var auth smtp.Auth
	if config.Username != "" {
		auth = smtp.PlainAuth("", config.Username, config.Password, config.Hostname)
	}

	return &smtpEmailSender{config: config, auth: auth}
}

// smtpSend is a package-level variable so tests can replace the actual
// smtp.SendMail with a stub. It defaults to smtp.SendMail.
var smtpSend = smtp.SendMail

// needsEncoding reports whether s contains any non-ASCII characters.
func needsEncoding(s string) bool {
	for _, r := range s {
		if r > unicode.MaxASCII {
			return true
		}
	}
	return false
}

// headerLineBreaks replaces the two characters that terminate a header. They
// are turned into spaces rather than dropped so the value stays readable.
var headerLineBreaks = strings.NewReplacer("\r", " ", "\n", " ")

// encodeHeader prepares a value for use as a header. CR and LF are removed
// first and unconditionally: a value carrying either would otherwise end the
// header and let the rest be read as further headers, and that must not depend
// on whether the value happens to be non-ASCII. What is left is Q-encoded per
// RFC 2047 when it needs to be, and returned unchanged when it does not.
//
// Address headers do not go through here — see addressHeader, which also has
// to quote characters that are legal in a name but structural in an address.
func encodeHeader(value string) string {
	value = headerLineBreaks.Replace(value)
	if !needsEncoding(value) {
		return value
	}
	return mime.QEncoding.Encode("utf-8", value)
}

// addressHeader formats a name and address as a single header value.
//
// mail.Address does the whole job: it encodes a non-ASCII name, quotes one
// containing characters that would otherwise be read as address syntax — a
// comma in "Müller, Hans" splits it into two recipients otherwise — and, since
// both CR and LF force the encoded form, leaves no way to break out of the
// header. The name is attacker-controlled: it is built from the first and last
// name of an account, which registration only trims.
func addressHeader(name, address string) string {
	return (&mail.Address{Name: headerLineBreaks.Replace(name), Address: address}).String()
}

func (s *smtpEmailSender) getAddress() string {
	return s.config.Hostname + ":" + strconv.Itoa(s.config.Port)
}

func (s *smtpEmailSender) SendMail(email string, displayName string, subject string, message string, isHTML bool, attachments map[string][]byte) error {
	receivers := []string{email}

	contentType := "text/plain"
	if isHTML {
		contentType = "text/html"
	}

	// buffer for mail
	buf := bytes.NewBuffer(nil)

	fmt.Fprintf(buf, "From: %s\r\n", addressHeader(s.config.SenderName, s.config.SenderEmail))
	fmt.Fprintf(buf, "Subject: %s\r\n", encodeHeader(subject))
	fmt.Fprintf(buf, "To: %s\r\n", addressHeader(displayName, email))
	buf.WriteString("MIME-Version: 1.0\r\n")

	if len(attachments) > 0 {
		if err := writeMultipartBody(buf, contentType, message, attachments); err != nil {
			return fmt.Errorf("building message for %q: %w", email, err)
		}
	} else {
		fmt.Fprintf(buf, "Content-Type: %s; charset=utf-8\r\n\r\n", contentType)
		buf.WriteString(message)
	}

	return smtpSend(s.getAddress(), s.auth, s.config.SenderEmail, receivers, buf.Bytes())
}

// writeMultipartBody writes the multipart/mixed body — the message itself
// followed by one base64 part per attachment — into buf.
func writeMultipartBody(buf *bytes.Buffer, contentType string, message string, attachments map[string][]byte) error {
	writer := multipart.NewWriter(buf)
	fmt.Fprintf(buf, "Content-Type: multipart/mixed; boundary=%s\r\n\r\n", writer.Boundary())

	textPartHeader := make(textproto.MIMEHeader)
	textPartHeader.Set("Content-Type", contentType+"; charset=utf-8")

	textPart, err := writer.CreatePart(textPartHeader)
	if err != nil {
		return fmt.Errorf("creating text part: %w", err)
	}
	if _, err := textPart.Write([]byte(message)); err != nil {
		return fmt.Errorf("writing text part: %w", err)
	}

	for name, content := range attachments {
		if err := writeAttachment(writer, name, content); err != nil {
			return fmt.Errorf("attaching %q: %w", name, err)
		}
	}

	// Writes the terminating boundary; the message is incomplete without it.
	if err := writer.Close(); err != nil {
		return fmt.Errorf("closing multipart writer: %w", err)
	}
	return nil
}

func writeAttachment(writer *multipart.Writer, name string, content []byte) error {
	// An unknown extension yields an empty media type, which is not a legal
	// header value; the RFC 2046 fallback for "unidentified" is octet-stream.
	mediaType := mime.TypeByExtension(filepath.Ext(name))
	if mediaType == "" {
		mediaType = "application/octet-stream"
	}

	// A filename is not necessarily a safe header parameter: quotes, semicolons
	// and line breaks all have meaning here. FormatMediaType quotes and encodes
	// it, and returns empty for a name it cannot represent at all — in which
	// case the attachment is still sent, just unnamed.
	disposition := mime.FormatMediaType("attachment", map[string]string{"filename": name})
	if disposition == "" {
		disposition = "attachment"
	}

	partHeader := make(textproto.MIMEHeader)
	partHeader.Set("Content-Type", mediaType)
	partHeader.Set("Content-Transfer-Encoding", "base64")
	partHeader.Set("Content-Disposition", disposition)

	part, err := writer.CreatePart(partHeader)
	if err != nil {
		return err
	}

	encoded := make([]byte, base64.StdEncoding.EncodedLen(len(content)))
	base64.StdEncoding.Encode(encoded, content)
	for i := 0; i < len(encoded); i += base64LineLength {
		end := min(i+base64LineLength, len(encoded))
		if _, err := part.Write(encoded[i:end]); err != nil {
			return err
		}
		if _, err := part.Write([]byte("\r\n")); err != nil {
			return err
		}
	}
	return nil
}

// consoleEmailSender is a development/test implementation that logs emails to stdout.
type consoleEmailSender struct{}

// NewConsoleEmailSender creates a new EmailSender that logs emails to the console.
func NewConsoleEmailSender() services.EmailSender {
	return &consoleEmailSender{}
}

func (c *consoleEmailSender) SendMail(email string, displayName string, subject string, message string, isHTML bool, attachments map[string][]byte) error {
	slog.Info("console email",
		"to", email,
		"displayName", displayName,
		"subject", subject,
		"isHTML", isHTML,
		"attachments", len(attachments),
		"message", message,
	)
	return nil
}
