package cli

import (
	"bytes"
	"context"
	"testing"

	"github.com/azuki774/myscrapers/myscraper/internal/scrape"
	"github.com/azuki774/myscrapers/myscraper/internal/smbccard"
)

type fakeRunner struct {
	result scrape.Result
	err    error
	req    scrape.Request
}

func (f *fakeRunner) Run(ctx context.Context, req scrape.Request) (scrape.Result, error) {
	f.req = req
	return f.result, f.err
}

// TestRunPrintsSavedPath verifies that cli.Run returns exit code 0
// on success, writes the expected "saved <path> (<title>)" line to
// stdout, and forwards the parsed --mode to the injected Runner.
func TestRunPrintsSavedPath(t *testing.T) {
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	runner := &fakeRunner{
		result: scrape.Result{
			Title:      "WEB明細",
			OutputPath: "tmp/smbc.html",
		},
	}

	exitCode := Run(
		[]string{"--mode", smbccard.ModeWebMeisaiHTMLDump, "--out", "tmp/smbc.html"},
		stdout,
		stderr,
		runner,
	)

	if exitCode != 0 {
		t.Fatalf("exitCode = %d, want 0; stderr = %q", exitCode, stderr.String())
	}
	if runner.req.Mode != smbccard.ModeWebMeisaiHTMLDump {
		t.Fatalf("Mode = %q, want %q", runner.req.Mode, smbccard.ModeWebMeisaiHTMLDump)
	}
	if stdout.String() != "saved tmp/smbc.html (WEB明細)\n" {
		t.Fatalf("stdout = %q", stdout.String())
	}
}
