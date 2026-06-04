package cli

import (
	"context"
	"fmt"
	"io"

	"github.com/azuki774/myscrapers/myscraper/internal/scrape"
)

// Runner is the contract the CLI depends on for executing a scrape.
// It is satisfied by scrape.Service in production and by fakes in
// tests, which lets the CLI be unit-tested without a real browser.
type Runner interface {
	Run(ctx context.Context, req scrape.Request) (scrape.Result, error)
}

// Run is the top-level entry point used by cmd/myscraper. It parses
// the given argv slice, dispatches to the injected Runner, and writes
// either a one-line "saved" success message to stdout or the error to
// stderr. The exit code is 0 on success, 1 on a runner error, and 2
// on argument parsing failure so callers can distinguish the two
// failure modes.
func Run(args []string, stdout, stderr io.Writer, runner Runner) int {
	opts, err := ParseArgs(args)
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 2
	}

	result, err := runner.Run(context.Background(), scrape.Request{
		URL:        opts.URL,
		OutputPath: opts.OutputPath,
		Headless:   opts.Headless,
	})
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}

	fmt.Fprintf(stdout, "saved %s (%s)\n", result.OutputPath, result.Title)
	return 0
}
