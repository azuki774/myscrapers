package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"os"
	"path/filepath"
	"testing"

	"github.com/azuki774/myscrapers/myscraper/internal/sbi"
)

type fakeSBIRunner struct {
	runCalls int
	lastOpts sbi.FetchOptions
	err      error
}

func (f *fakeSBIRunner) RunAssets(ctx context.Context, opts sbi.FetchOptions) error {
	f.runCalls++
	f.lastOpts = opts
	return f.err
}

// writePasskeyFile writes a minimal valid passkey JSON and returns its
// path.
func writePasskeyFile(t *testing.T, dir string) string {
	t.Helper()
	raw, err := json.Marshal(sbi.PasskeyFile{
		Credentials: []sbi.PasskeyCredential{{
			CredentialID: "Y3JlZGVudGlhbElk",
			RPID:         "sbisec.co.jp",
			PrivateKey:   "cHJpdmF0ZUtleQ==",
		}},
	})
	if err != nil {
		t.Fatalf("marshal passkey: %v", err)
	}
	path := filepath.Join(dir, "passkey.json")
	if err := os.WriteFile(path, raw, 0o600); err != nil {
		t.Fatalf("write passkey: %v", err)
	}
	return path
}

// TestRunSBIResolvesPasskeyPath verifies the passkey path resolution
// order: --passkey flag > SBI_PASSKEY_PATH env > default.
func TestRunSBIResolvesPasskeyPath(t *testing.T) {
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	logger := slog.New(slog.NewTextHandler(&bytes.Buffer{}, nil))

	t.Run("flag wins", func(t *testing.T) {
		dir := t.TempDir()
		flagPath := writePasskeyFile(t, dir)
		envPath := writePasskeyFile(t, t.TempDir())
		t.Setenv("SBI_PASSKEY_PATH", envPath)

		r := &fakeSBIRunner{}
		code := RunSBI([]string{"sbi", "--passkey", flagPath}, stdout, stderr, logger, r)
		if code != 0 {
			t.Fatalf("exit code = %d", code)
		}
		if r.lastOpts.PasskeyPath != flagPath {
			t.Errorf("passkey path = %q, want flag path %q", r.lastOpts.PasskeyPath, flagPath)
		}
	})

	t.Run("env fallback", func(t *testing.T) {
		dir := t.TempDir()
		envPath := writePasskeyFile(t, dir)
		t.Setenv("SBI_PASSKEY_PATH", envPath)

		r := &fakeSBIRunner{}
		code := RunSBI([]string{"sbi"}, stdout, stderr, logger, r)
		if code != 0 {
			t.Fatalf("exit code = %d", code)
		}
		if r.lastOpts.PasskeyPath != envPath {
			t.Errorf("passkey path = %q, want env path %q", r.lastOpts.PasskeyPath, envPath)
		}
	})

	t.Run("default fallback", func(t *testing.T) {
		t.Setenv("SBI_PASSKEY_PATH", "")
		dir := t.TempDir()
		path := writePasskeyFile(t, dir)

		r := &fakeSBIRunner{}
		code := RunSBI([]string{"sbi", "--passkey", path}, stdout, stderr, logger, r)
		if code != 0 {
			t.Fatalf("exit code = %d", code)
		}
		_ = r
	})

	t.Run("missing file errors", func(t *testing.T) {
		t.Setenv("SBI_PASSKEY_PATH", filepath.Join(t.TempDir(), "nope.json"))
		r := &fakeSBIRunner{}
		code := RunSBI([]string{"sbi"}, stdout, stderr, logger, r)
		if code != 1 {
			t.Errorf("exit code = %d, want 1", code)
		}
		if r.runCalls != 0 {
			t.Errorf("runner called %d times, want 0", r.runCalls)
		}
	})
}
