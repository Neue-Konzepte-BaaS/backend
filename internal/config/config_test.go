package config

import (
	"strings"
	"testing"
)

func TestParseIntEnv(t *testing.T) {
	tests := []struct {
		name     string
		set      bool
		value    string
		fallback int
		want     int
		wantErr  bool
	}{
		{name: "unset uses the fallback", set: false, fallback: 587, want: 587},
		{name: "empty uses the fallback", set: true, value: "", fallback: 587, want: 587},
		{name: "parses a value", set: true, value: "2525", fallback: 587, want: 2525},
		{name: "parses a negative value", set: true, value: "-1", fallback: 587, want: -1},
		{name: "rejects a non-number", set: true, value: "not-a-port", fallback: 587, wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			const key = "TEST_INT_ENV"
			if tt.set {
				t.Setenv(key, tt.value)
			}

			got, err := parseIntEnv(key, tt.fallback)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("expected an error for %q", tt.value)
				}
				if !strings.Contains(err.Error(), key) {
					t.Errorf("error = %v, want it to name the variable", err)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tt.want {
				t.Errorf("got = %d, want %d", got, tt.want)
			}
		})
	}
}

// validSMTPConfig is the smallest config that passes SMTP validation with
// sending switched on.
func validSMTPConfig() Config {
	return Config{
		SMTPEnabled:     true,
		SMTPHost:        "smtp.example.com",
		SMTPPort:        587,
		SMTPSenderName:  "BaaS",
		SMTPSenderEmail: "noreply@example.com",
	}
}

func TestValidateSMTP_ValidConfigPasses(t *testing.T) {
	if errs := validateSMTP(validSMTPConfig()); len(errs) != 0 {
		t.Errorf("unexpected errors: %v", errs)
	}
}

func TestValidateSMTP_DisabledSkipsTheCredentialRules(t *testing.T) {
	// Nothing but the port is configured, which is the default state of a
	// developer checkout; it must not stop the process from starting.
	c := Config{SMTPEnabled: false, SMTPPort: 587}

	if errs := validateSMTP(c); len(errs) != 0 {
		t.Errorf("unexpected errors with SMTP disabled: %v", errs)
	}
}

func TestValidateSMTP_PortIsCheckedEvenWhenDisabled(t *testing.T) {
	c := Config{SMTPEnabled: false, SMTPPort: 70000}

	errs := validateSMTP(c)
	if len(errs) != 1 {
		t.Fatalf("got %d errors, want 1: %v", len(errs), errs)
	}
	if !strings.Contains(errs[0].Error(), "SMTPPort") {
		t.Errorf("error = %v, want it to name SMTPPort", errs[0])
	}
}

func TestValidateSMTP_EnabledRequiresHostAndSender(t *testing.T) {
	tests := []struct {
		name    string
		mutate  func(*Config)
		wantErr string
	}{
		{
			name:    "missing host",
			mutate:  func(c *Config) { c.SMTPHost = "" },
			wantErr: "SMTPHost",
		},
		{
			name:    "missing sender name",
			mutate:  func(c *Config) { c.SMTPSenderName = "" },
			wantErr: "SMTPSenderName",
		},
		{
			name:    "missing sender email",
			mutate:  func(c *Config) { c.SMTPSenderEmail = "" },
			wantErr: "SMTPSenderEmail",
		},
		{
			name:    "malformed sender email",
			mutate:  func(c *Config) { c.SMTPSenderEmail = "not-an-address" },
			wantErr: "SMTPSenderEmail",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := validSMTPConfig()
			tt.mutate(&c)

			errs := validateSMTP(c)
			if len(errs) != 1 {
				t.Fatalf("got %d errors, want 1: %v", len(errs), errs)
			}
			if !strings.Contains(errs[0].Error(), tt.wantErr) {
				t.Errorf("error = %v, want it to name %s", errs[0], tt.wantErr)
			}
		})
	}
}

func TestValidateSMTP_ReportsEveryMissingFieldAtOnce(t *testing.T) {
	// Validate collects rather than short-circuits, so a misconfigured deploy
	// learns about all of its problems in one startup attempt.
	c := Config{SMTPEnabled: true, SMTPPort: 587}

	if errs := validateSMTP(c); len(errs) != 3 {
		t.Errorf("got %d errors, want 3 (host, sender name, sender email): %v", len(errs), errs)
	}
}
