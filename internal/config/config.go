package config

import (
	"errors"
	"fmt"
	"net/url"
)

type Config struct {
	DatabaseURL   string
	DBAutoMigrate bool

	JWTSecret string

	SameSiteStrict bool
	CORSEnabled    bool
	FrontendURL    string
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
