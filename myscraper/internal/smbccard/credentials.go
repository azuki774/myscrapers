// Package smbccard owns SMBC Card (Vpass) specific automation. It
// loads the member's Vpass credentials from the environment and runs
// the browser flow that reaches the WEB明細 page, but it stays
// independent of any concrete browser implementation: the workflow
// talks to a small InteractivePage interface so unit tests can drive
// it with an in-memory fake.
package smbccard

import (
	"errors"
	"os"
)

// ModeWebMeisaiHTMLDump identifies the CLI mode that logs in to Vpass
// and saves the WEB明細 page HTML snapshot. It is also used by the CLI
// to validate the --mode flag value.
const ModeWebMeisaiHTMLDump = "smbc-card-webmeisai"

// Credentials is the minimum set of secrets needed to log in to Vpass.
type Credentials struct {
	LoginID  string
	Password string
}

// LoadCredentialsFromEnv returns the Vpass credentials, preferring
// the dedicated SMBC_VPASS_ID/SMBC_VPASS_PASSWORD pair and falling
// back to the legacy user/pass pair. It returns an error when both
// pairs are absent or empty, so callers can surface a clear message
// to the operator instead of silently submitting blank fields.
func LoadCredentialsFromEnv() (Credentials, error) {
	loginID := envOrFallback("SMBC_VPASS_ID", "user")
	password := envOrFallback("SMBC_VPASS_PASSWORD", "pass")
	if loginID == "" || password == "" {
		return Credentials{}, errors.New("SMBC credentials are required: set SMBC_VPASS_ID/SMBC_VPASS_PASSWORD or user/pass")
	}
	return Credentials{
		LoginID:  loginID,
		Password: password,
	}, nil
}

// envOrFallback returns the value of the primary env var when it is
// non-empty, otherwise the value of the fallback env var. The second
// return value of os.LookupEnv is intentionally ignored: an unset
// env var is treated the same as an empty one for the purposes of
// picking a credential source.
func envOrFallback(primary, fallback string) string {
	if value := os.Getenv(primary); value != "" {
		return value
	}
	return os.Getenv(fallback)
}
