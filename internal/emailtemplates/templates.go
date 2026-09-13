// Package emailtemplates holds the HTML bodies used for outgoing mail.
//
// They are embedded rather than read from disk so the binary is self-contained:
// the runtime container image copies only the binary and the migrations, and a
// missing template directory would otherwise turn every notification into a
// runtime failure.
package emailtemplates

import "embed"

// FS holds every mail template, named without the .html suffix when passed to
// NotificationService.SendMailFromTemplate.
//
//go:embed *.html
var FS embed.FS
