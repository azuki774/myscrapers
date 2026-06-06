package cli

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/azuki774/myscrapers/myscraper/internal/moneyforward"
)

type fakeMoneyforwardRunner struct {
	fetchCalls  int
	updateCalls int
	lastFetchS3 bool
	err         error
}

func (f *fakeMoneyforwardRunner) RunFetch(ctx context.Context, opts moneyforward.FetchOptions, s3Upload bool) error {
	f.fetchCalls++
	f.lastFetchS3 = s3Upload
	return f.err
}

func (f *fakeMoneyforwardRunner) RunUpdate(ctx context.Context, opts moneyforward.UpdateOptions) error {
	f.updateCalls++
	return f.err
}

func TestRunMoneyforwardRequiresExactlyOneVerb(t *testing.T) {
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	r := &fakeMoneyforwardRunner{}

	cases := []struct {
		name string
		args []string
		want int
	}{
		{"no verb", []string{"moneyforward"}, 2},
		{"both verbs", []string{"moneyforward", "--fetch", "--update"}, 2},
		{"fetch", []string{"moneyforward", "--fetch"}, 0},
		{"update", []string{"moneyforward", "--update"}, 0},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			cookiePath := filepath.Join(dir, "cookie.json")
			if err := os.WriteFile(cookiePath, []byte(`[{"name":"sid","value":"abc","domain":".moneyforward.com","path":"/","sameSite":"lax"}]`), 0o600); err != nil {
				t.Fatalf("write cookie: %v", err)
			}
			t.Setenv("MF_COOKIE_PATH", cookiePath)

			stdout.Reset()
			stderr.Reset()
			code := RunMoneyforward(tc.args, stdout, stderr, slog.New(slog.NewTextHandler(stderr, nil)), r)
			if code != tc.want {
				t.Fatalf("exit = %d, want %d; stderr = %q", code, tc.want, stderr.String())
			}
		})
	}
}

func TestRunMoneyforwardPropagatesRunnerErrorAndS3Flag(t *testing.T) {
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	r := &fakeMoneyforwardRunner{err: errors.New("boom")}

	dir := t.TempDir()
	cookiePath := filepath.Join(dir, "cookie.json")
	if err := os.WriteFile(cookiePath, []byte(`[{"name":"sid","value":"abc","domain":".moneyforward.com","path":"/","sameSite":"lax"}]`), 0o600); err != nil {
		t.Fatalf("write cookie: %v", err)
	}
	t.Setenv("MF_COOKIE_PATH", cookiePath)

	code := RunMoneyforward([]string{"moneyforward", "--fetch", "--s3-upload"}, stdout, stderr, slog.New(slog.NewTextHandler(stderr, nil)), r)
	if code != 1 {
		t.Fatalf("exit = %d, want 1; stderr = %q", code, stderr.String())
	}
	if r.fetchCalls != 1 {
		t.Fatalf("fetchCalls = %d, want 1", r.fetchCalls)
	}
	if !r.lastFetchS3 {
		t.Fatalf("lastFetchS3 = false, want true")
	}
}

func TestRunMoneyforwardWarnsOnS3UploadWithUpdate(t *testing.T) {
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	r := &fakeMoneyforwardRunner{}

	dir := t.TempDir()
	cookiePath := filepath.Join(dir, "cookie.json")
	if err := os.WriteFile(cookiePath, []byte(`[{"name":"sid","value":"abc","domain":".moneyforward.com","path":"/","sameSite":"lax"}]`), 0o600); err != nil {
		t.Fatalf("write cookie: %v", err)
	}
	t.Setenv("MF_COOKIE_PATH", cookiePath)

	code := RunMoneyforward([]string{"moneyforward", "--update", "--s3-upload"}, stdout, stderr, slog.New(slog.NewTextHandler(stderr, nil)), r)
	if code != 0 {
		t.Fatalf("exit = %d, want 0; stderr = %q", code, stderr.String())
	}
	if r.updateCalls != 1 {
		t.Fatalf("updateCalls = %d, want 1", r.updateCalls)
	}
	if !strings.Contains(stderr.String(), "--s3-upload") {
		t.Fatalf("stderr = %q, want a warning mentioning --s3-upload", stderr.String())
	}
}
