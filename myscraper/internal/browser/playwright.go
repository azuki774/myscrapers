// Package browser holds the Playwright-backed implementation of
// scrape.Browser. It owns the Playwright driver download path,
// the chromium executable lookup, and the per-request browser
// lifecycle. The package is kept small so the scrape service can
// stay free of Playwright-specific types.
package browser

import (
	"context"
	"fmt"
	"os"

	"github.com/azuki774/myscrapers/myscraper/internal/chromium"
	"github.com/azuki774/myscrapers/myscraper/internal/scrape"
	"github.com/mxschmitt/playwright-go"
)

// PlaywrightBrowser is a zero-value, stateless adapter that satisfies
// scrape.Browser. It is safe to construct at program start and
// reuse across requests; per-request state is created by Playwright
// itself.
type PlaywrightBrowser struct{}

// InstallDriver downloads (or refreshes) the Playwright Node driver
// into the directory resolved by runOptions. It is safe to call on
// every Fetch because the underlying installer is idempotent.
func InstallDriver() error {
	return playwright.Install(runOptions())
}

// runOptions centralizes the Playwright driver configuration: the
// driver lives under PLAYWRIGHT_DRIVER_PATH when set, otherwise under
// the repo-local .playwright-driver directory; browser binaries are
// not managed here because the dev shell supplies chromium directly.
func runOptions() *playwright.RunOptions {
	driverDirectory := os.Getenv("PLAYWRIGHT_DRIVER_PATH")
	if driverDirectory == "" {
		driverDirectory = ".playwright-driver"
	}
	return &playwright.RunOptions{
		DriverDirectory:     driverDirectory,
		SkipInstallBrowsers: true,
	}
}

// Fetch opens url in a headless (or headed) Chromium driven by
// Playwright and returns the rendered page snapshot. The browser
// process is started on each call and torn down before returning so
// callers do not need to manage Playwright's lifecycle directly.
func (PlaywrightBrowser) Fetch(ctx context.Context, url string, headless bool) (scrape.PageSnapshot, error) {
	if err := InstallDriver(); err != nil {
		return scrape.PageSnapshot{}, fmt.Errorf("install playwright driver: %w", err)
	}

	pw, err := playwright.Run(runOptions())
	if err != nil {
		return scrape.PageSnapshot{}, fmt.Errorf("start playwright: %w", err)
	}
	defer pw.Stop()

	executablePath, err := chromium.ExecutablePath()
	if err != nil {
		return scrape.PageSnapshot{}, err
	}

	browser, err := pw.Chromium.Launch(playwright.BrowserTypeLaunchOptions{
		ExecutablePath: playwright.String(executablePath),
		Headless:       playwright.Bool(headless),
	})
	if err != nil {
		return scrape.PageSnapshot{}, fmt.Errorf("launch chromium: %w", err)
	}
	defer browser.Close()

	page, err := browser.NewPage()
	if err != nil {
		return scrape.PageSnapshot{}, fmt.Errorf("new page: %w", err)
	}

	if _, err := page.Goto(url); err != nil {
		return scrape.PageSnapshot{}, fmt.Errorf("goto %s: %w", url, err)
	}

	title, err := page.Title()
	if err != nil {
		return scrape.PageSnapshot{}, fmt.Errorf("read title: %w", err)
	}

	html, err := page.Content()
	if err != nil {
		return scrape.PageSnapshot{}, fmt.Errorf("read content: %w", err)
	}

	return scrape.PageSnapshot{
		URL:   url,
		Title: title,
		HTML:  html,
	}, nil
}
