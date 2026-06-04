package smbccard

import (
	"os"
	"strings"
	"testing"
)

func TestLoadCredentialsFromEnv(t *testing.T) {
	t.Run("prefers dedicated env vars", func(t *testing.T) {
		t.Setenv("SMBC_VPASS_ID", "dedicated-id")
		t.Setenv("SMBC_VPASS_PASSWORD", "dedicated-pass")
		t.Setenv("user", "legacy-id")
		t.Setenv("pass", "legacy-pass")

		got, err := LoadCredentialsFromEnv()
		if err != nil {
			t.Fatalf("LoadCredentialsFromEnv() error = %v", err)
		}
		if got.LoginID != "dedicated-id" || got.Password != "dedicated-pass" {
			t.Fatalf("credentials = %#v", got)
		}
	})

	t.Run("falls back to legacy env vars", func(t *testing.T) {
		t.Setenv("SMBC_VPASS_ID", "")
		t.Setenv("SMBC_VPASS_PASSWORD", "")
		t.Setenv("user", "legacy-id")
		t.Setenv("pass", "legacy-pass")

		got, err := LoadCredentialsFromEnv()
		if err != nil {
			t.Fatalf("LoadCredentialsFromEnv() error = %v", err)
		}
		if got.LoginID != "legacy-id" || got.Password != "legacy-pass" {
			t.Fatalf("credentials = %#v", got)
		}
	})

	t.Run("rejects missing credentials", func(t *testing.T) {
		t.Setenv("SMBC_VPASS_ID", "")
		t.Setenv("SMBC_VPASS_PASSWORD", "")
		t.Setenv("user", "")
		t.Setenv("pass", "")

		_, err := LoadCredentialsFromEnv()
		if err == nil || err.Error() != "SMBC credentials are required: set SMBC_VPASS_ID/SMBC_VPASS_PASSWORD or user/pass" {
			t.Fatalf("expected missing credentials error, got %v", err)
		}
	})
}

func TestEnvOrFallback(t *testing.T) {
	t.Setenv("PRIMARY", "")
	t.Setenv("FALLBACK", "fallback-value")

	if got := envOrFallback("PRIMARY", "FALLBACK"); got != "fallback-value" {
		t.Fatalf("envOrFallback() = %q, want %q", got, "fallback-value")
	}

	if _, ok := os.LookupEnv("PRIMARY"); !ok {
		t.Fatalf("PRIMARY should still exist in environment during the test")
	}
}

func TestCredentialsStringMasksPassword(t *testing.T) {
	c := Credentials{LoginID: "member-id", Password: "super-secret"}
	got := c.String()
	if !strings.Contains(got, "member-id") {
		t.Fatalf("String() = %q, want to contain LoginID", got)
	}
	if strings.Contains(got, "super-secret") {
		t.Fatalf("String() = %q, must not contain password", got)
	}
}
