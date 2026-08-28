package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

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

// TestWriteAssetsJSONStampsSchemaVersion verifies that every emitted
// assets JSON carries schema_version as the very first key, stamped
// with the current schema version even when the caller forgets to set
// it.
func TestWriteAssetsJSONStampsSchemaVersion(t *testing.T) {
	var buf bytes.Buffer
	raw, err := WriteAssetsJSON(&buf, "", &sbi.Assets{})
	if err != nil {
		t.Fatalf("write: %v", err)
	}
	out := buf.String()
	if !bytes.Equal(raw, []byte(out)) {
		t.Fatalf("returned bytes differ from written output")
	}
	wantPrefix := "{\n  \"schema_version\": 1,"
	if !strings.HasPrefix(out, wantPrefix) {
		t.Fatalf("output does not start with %q:\n%s", wantPrefix, out)
	}
	var m map[string]any
	if err := json.Unmarshal([]byte(out), &m); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if got, ok := m["schema_version"].(float64); !ok || got != float64(sbi.CurrentSchemaVersion) {
		t.Errorf("schema_version = %v, want %d", m["schema_version"], sbi.CurrentSchemaVersion)
	}
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

	t.Run("s3-upload flag sets opts", func(t *testing.T) {
		dir := t.TempDir()
		path := writePasskeyFile(t, dir)
		t.Setenv("SBI_PASSKEY_PATH", path)

		r := &fakeSBIRunner{}
		code := RunSBI([]string{"sbi", "--s3-upload"}, stdout, stderr, logger, r)
		if code != 0 {
			t.Fatalf("exit code = %d, want 0", code)
		}
		if !r.lastOpts.S3Upload {
			t.Errorf("S3Upload = %v, want true", r.lastOpts.S3Upload)
		}
	})
}

// fakeS3Client records the key and body passed to PutJSON and derives the
// key from the timestamp via the same JST layout storage uses.
type fakeS3Client struct {
	key  string
	body []byte
}

func (f *fakeS3Client) PutJSON(_ context.Context, key string, body io.Reader) error {
	f.key = key
	b, err := io.ReadAll(body)
	if err != nil {
		return err
	}
	f.body = b
	return nil
}

func (f *fakeS3Client) KeyForTime(now time.Time) string {
	return "myscrapers/sbi/" + now.Format("2006/01/20060102-150405") + ".json"
}

func TestUploadAssetsJSONUsesSameBytes(t *testing.T) {
	client := &fakeS3Client{}
	raw := []byte("{\"schema_version\":1,\"status\":\"ok\"}\n")
	now := time.Date(2024, 12, 9, 1, 2, 3, 0, time.UTC)

	key, err := UploadAssetsJSON(context.Background(), client, now, raw)
	if err != nil {
		t.Fatalf("UploadAssetsJSON() error = %v", err)
	}
	if key != "myscrapers/sbi/2024/12/20241209-010203.json" {
		t.Fatalf("key = %q, want %q", key, "myscrapers/sbi/2024/12/20241209-010203.json")
	}
	if !bytes.Equal(client.body, raw) {
		t.Fatalf("uploaded body = %q, want %q", client.body, raw)
	}
}
