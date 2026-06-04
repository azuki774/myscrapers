// Package scrape defines the Browser interface, request/result
// payloads, and a Service that drives a Browser and writes its HTML
// snapshot to disk. The CLI layer talks to Service through these
// types so the concrete browser implementation (Playwright or a test
// fake) can be swapped without changing the CLI.
package scrape

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
)

// Browser is the minimum capability the scrape service needs: open a
// URL with a real browser engine and return the rendered snapshot.
// Implementations live in package browser; tests use a fake.
type Browser interface {
	Fetch(ctx context.Context, url string, headless bool) (PageSnapshot, error)
}

// PageSnapshot is the result of fetching a single page: where the
// browser ended up (URL may differ from the requested URL after
// redirects), the rendered <title>, and the full HTML body.
type PageSnapshot struct {
	URL   string
	Title string
	HTML  string
}

// Request is the input to Service.Run: the URL to fetch, the file the
// HTML should be written to, and whether the browser should run
// without a visible window.
type Request struct {
	URL        string
	OutputPath string
	Headless   bool
}

// Result is the success view returned to the caller. Only the data
// the CLI needs to print is included; the full HTML stays on disk.
type Result struct {
	Title      string
	OutputPath string
}

// Service runs a single scrape request end-to-end: delegate the
// browser work to a Browser, ensure the output directory exists,
// and write the HTML snapshot to disk. It is intentionally a
// dependency-injection-friendly value type so the CLI can hand it
// any Browser implementation.
type Service struct {
	Browser Browser
}

// Run executes a Request. It returns an error if no Browser has been
// configured, the Browser fails, or the output cannot be written.
// The output file is created with mode 0644; callers that handle
// sensitive HTML (e.g. authenticated financial statements) should
// not reuse this path and must tighten permissions themselves.
func (s Service) Run(ctx context.Context, req Request) (Result, error) {
	if s.Browser == nil {
		return Result{}, fmt.Errorf("browser is required")
	}

	snapshot, err := s.Browser.Fetch(ctx, req.URL, req.Headless)
	if err != nil {
		return Result{}, err
	}

	// Create the parent directory before WriteFile so callers can pass
	// paths under not-yet-existing tmp/-style directories.
	if err := os.MkdirAll(filepath.Dir(req.OutputPath), 0o755); err != nil {
		return Result{}, err
	}
	if err := os.WriteFile(req.OutputPath, []byte(snapshot.HTML), 0o644); err != nil {
		return Result{}, err
	}

	return Result{
		Title:      snapshot.Title,
		OutputPath: req.OutputPath,
	}, nil
}
