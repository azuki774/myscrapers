package cli

import (
	"bytes"
	"context"
	"log/slog"
	"testing"

	"github.com/azuki774/myscrapers/myscraper/internal/scrape"
)

type fakeRunner struct {
	result scrape.Result
	err    error
}

func (f fakeRunner) Run(ctx context.Context, req scrape.Request) (scrape.Result, error) {
	return f.result, f.err
}

func TestRunPrintsSavedPath(t *testing.T) {
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	logger := slog.New(slog.NewJSONHandler(stderr, nil))

	exitCode := Run(
		[]string{"--url", "https://github.com", "--out", "tmp/github.html"},
		stdout,
		stderr,
		logger,
		fakeRunner{
			result: scrape.Result{
				Title:      "GitHub",
				OutputPath: "tmp/github.html",
			},
		},
	)

	if exitCode != 0 {
		t.Fatalf("exitCode = %d, want 0; stderr = %q", exitCode, stderr.String())
	}
}
