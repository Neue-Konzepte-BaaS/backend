package config

import (
	"errors"
	"fmt"
	"net/mail"
	"net/url"
	"os"
	"strconv"
)

type Config struct {
	DatabaseURL   string
	DBAutoMigrate bool

	JWTSecret string

	SameSiteStrict bool
	CookieSecure   bool
	CORSEnabled    bool
	FrontendURL    string

	SMTPEnabled     bool
	SMTPHost        string
	SMTPPort        int
	SMTPUsername    string
	SMTPPassword    string
	SMTPSenderName  string
	SMTPSenderEmail string
}

func Load() (Config, error) {
	dbAutoMigrate, err := parseBoolEnv("DB_AUTO_MIGRATE", false)
	if err != nil {
		return Config{}, err
	}
	sameSiteStrict, err := parseBoolEnv("SAME_SITE_STRICT", true)
	if err != nil {
		return Config{}, err
	}
	corsEnabled, err := parseBoolEnv("CORS_ENABLED", false)
	if err != nil {
		return Config{}, err
	}
	// Defaults to true so a misconfigured production deploy fails closed;
	// set COOKIE_SECURE=false for local http development.
	cookieSecure, err := parseBoolEnv("COOKIE_SECURE", true)
	if err != nil {
		return Config{}, err
	}

	// Off by default so a developer who has not configured a relay still gets
	// a working server: main.go then logs notifications instead of sending.
	smtpEnabled, err := parseBoolEnv("SMTP_ENABLED", false)
	if err != nil {
		return Config{}, err
	}
	smtpPort, err := parseIntEnv("SMTP_PORT", 587)
	if err != nil {
		return Config{}, err
	}

	c := Config{
		DatabaseURL:   os.Getenv("DATABASE_URL"),
		DBAutoMigrate: dbAutoMigrate,

		JWTSecret: os.Getenv("JWT_SECRET"),

		SameSiteStrict: sameSiteStrict,
		CookieSecure:   cookieSecure,
		CORSEnabled:    corsEnabled,
		FrontendURL:    os.Getenv("FRONTEND_URL"),

		SMTPEnabled:     smtpEnabled,
		SMTPHost:        os.Getenv("SMTP_HOST"),
		SMTPPort:        smtpPort,
		SMTPUsername:    os.Getenv("SMTP_USERNAME"),
		SMTPPassword:    os.Getenv("SMTP_PASSWORD"),
		SMTPSenderName:  os.Getenv("SMTP_SENDER_NAME"),
		SMTPSenderEmail: os.Getenv("SMTP_SENDER_EMAIL"),
	}

	if err := c.Validate(); err != nil {
		return Config{}, err
	}

	return c, nil
}

func parseBoolEnv(key string, fallback bool) (bool, error) {
	raw, ok := os.LookupEnv(key)
	if !ok || raw == "" {
		return fallback, nil
	}
	v, err := strconv.ParseBool(raw)
	if err != nil {
		return false, fmt.Errorf("%s: %w", key, err)
	}
	return v, nil
}

func parseIntEnv(key string, fallback int) (int, error) {
	raw, ok := os.LookupEnv(key)
	if !ok || raw == "" {
		return fallback, nil
	}
	v, err := strconv.Atoi(raw)
	if err != nil {
		return 0, fmt.Errorf("%s: %w", key, err)
	}
	return v, nil
}

func (c Config) Validate() error {
	var errs []error

	if err := validateDatabaseURL(c.DatabaseURL); err != nil {
		errs = append(errs, err)
	}
	if err := validateJWTSecret(c.JWTSecret); err != nil {
		errs = append(errs, err)
	}
	if err := validateFrontendURL(c.FrontendURL, c.CORSEnabled); err != nil {
		errs = append(errs, err)
	}
	errs = append(errs, validateSMTP(c)...)

	return errors.Join(errs...)
}

func validateDatabaseURL(raw string) error {
	if raw == "" {
		return errors.New("DatabaseURL: must not be empty")
	}
	u, err := url.Parse(raw)
	if err != nil {
		return fmt.Errorf("DatabaseURL: %w", err)
	}
	if u.Scheme == "" {
		return errors.New("DatabaseURL: must include a scheme")
	}
	return nil
}

func validateJWTSecret(secret string) error {
	const minLen = 32
	if len(secret) < minLen {
		return fmt.Errorf("JWTSecret: must be at least %d characters", minLen)
	}
	return nil
}

func validateFrontendURL(raw string, corsEnabled bool) error {
	if raw == "" {
		if corsEnabled {
			return errors.New("FrontendURL: must not be empty when CORSEnabled is true")
		}
		return nil
	}
	u, err := url.Parse(raw)
	if err != nil {
		return fmt.Errorf("FrontendURL: %w", err)
	}
	if u.Scheme == "" || u.Host == "" {
		return errors.New("FrontendURL: must be an absolute URL")
	}
	return nil
}

// validateSMTP checks the mail settings. They are only required once
// SMTPEnabled is set: with it off the server wires a console sender, so an
// unconfigured relay must not stop the process from starting. The port is
// checked either way, because a nonsense value is a mistake worth reporting
// even when it is currently unused.
func validateSMTP(c Config) []error {
	var errs []error

	if c.SMTPPort < 1 || c.SMTPPort > 65535 {
		errs = append(errs, fmt.Errorf("SMTPPort: must be between 1 and 65535, got %d", c.SMTPPort))
	}

	if !c.SMTPEnabled {
		return errs
	}

	if c.SMTPHost == "" {
		errs = append(errs, errors.New("SMTPHost: must not be empty when SMTPEnabled is true"))
	}
	if c.SMTPSenderName == "" {
		errs = append(errs, errors.New("SMTPSenderName: must not be empty when SMTPEnabled is true"))
	}
	if c.SMTPSenderEmail == "" {
		errs = append(errs, errors.New("SMTPSenderEmail: must not be empty when SMTPEnabled is true"))
	} else if _, err := mail.ParseAddress(c.SMTPSenderEmail); err != nil {
		errs = append(errs, fmt.Errorf("SMTPSenderEmail: %w", err))
	}

	return errs
}
