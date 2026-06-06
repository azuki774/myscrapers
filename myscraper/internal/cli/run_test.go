package cli

import (
	"bytes"
	"context"
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

// TestRunPrintsSavedPath verifies that cli.Run returns exit code 0
// on success and writes the expected "saved <path> (<title>)" line
// to stdout when the injected Runner reports a successful Result.
func TestRunPrintsSavedPath(t *testing.T) {
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}

	exitCode := Run(
		[]string{"--url", "https://github.com", "--out", "tmp/github.html"},
		stdout,
		stderr,
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
	if stdout.String() != "saved tmp/github.html (GitHub)\n" {
		t.Fatalf("stdout = %q", stdout.String())
	}
}
