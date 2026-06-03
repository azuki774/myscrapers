package browser

import (
	"context"
	"fmt"
	"os"
	"os/exec"

	"github.com/azuki774/myscrapers/myscraper/internal/scrape"
	"github.com/playwright-community/playwright-go"
)

type PlaywrightBrowser struct{}

func InstallDriver() error {
	return playwright.Install(runOptions())
}

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

func chromiumExecutablePath() (string, error) {
	for _, candidate := range []string{"chromium", "chromium-browser", "google-chrome", "google-chrome-stable"} {
		if path, err := exec.LookPath(candidate); err == nil {
			return path, nil
		}
	}
	return "", fmt.Errorf("no chromium-compatible browser found in PATH")
}

func (PlaywrightBrowser) Fetch(ctx context.Context, url string, headless bool) (scrape.PageSnapshot, error) {
	if err := InstallDriver(); err != nil {
		return scrape.PageSnapshot{}, fmt.Errorf("install playwright driver: %w", err)
	}

	pw, err := playwright.Run(runOptions())
	if err != nil {
		return scrape.PageSnapshot{}, fmt.Errorf("start playwright: %w", err)
	}
	defer pw.Stop()

	executablePath, err := chromiumExecutablePath()
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
