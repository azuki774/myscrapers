// Package browser holds the Playwright-backed implementation of
// scrape.Browser. It owns the Playwright driver download path, the
// chromium executable lookup, and the per-request browser lifecycle.
// The package is kept small so the scrape service and the per-flow
// automation packages (e.g. internal/smbccard) can stay free of
// Playwright-specific types. The package also adapts a Playwright
// page to the smbccard.InteractivePage interface so the SMBC
// workflow can be unit-tested without a real browser.
package browser

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/azuki774/myscrapers/myscraper/internal/scrape"
	"github.com/azuki774/myscrapers/myscraper/internal/smbccard"
	"github.com/playwright-community/playwright-go"
)

// PlaywrightBrowser is a zero-value, stateless adapter that satisfies
// scrape.Browser. It is safe to construct at program start and reuse
// across requests; per-request state is created by Playwright itself.
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

// chromiumExecutablePath resolves the chromium binary that Playwright
// should drive. The dev shell puts a system chromium on PATH, so we
// look it up by name rather than bundling a browser. Order matches
// the most common package names on Debian, Nix, and macOS.
func chromiumExecutablePath() (string, error) {
	for _, candidate := range []string{"chromium", "chromium-browser", "google-chrome", "google-chrome-stable"} {
		if path, err := exec.LookPath(candidate); err == nil {
			return path, nil
		}
	}
	return "", fmt.Errorf("no chromium-compatible browser found in PATH")
}

// chromiumLaunchArgs returns the extra command-line arguments that
// should be passed to chromium on launch. They come from the
// MYSCRAPER_CHROMIUM_ARGS environment variable, which is split on
// any run of unicode whitespace. The variable is an escape hatch
// for environments where the default sandboxed launch fails (e.g.
// running as root in a container, /dev/shm too small, or running
// inside the nix dev shell where chromium's SUID sandbox cannot
// start). When the variable is unset or whitespace-only, the
// returned slice is nil so the Launch options stay minimal.
//
// Example: MYSCRAPER_CHROMIUM_ARGS="--no-sandbox --disable-dev-shm-usage"
func chromiumLaunchArgs() []string {
	raw := os.Getenv("MYSCRAPER_CHROMIUM_ARGS")
	if raw == "" {
		return nil
	}
	fields := strings.Fields(raw)
	if len(fields) == 0 {
		return nil
	}
	return fields
}

// defaultUserAgent is the User-Agent string used for every Playwright
// context opened by this package. It is a real Chrome 148 UA on
// Linux x86_64 — chosen to match the chromium binary version this
// repo's dev shell ships, which avoids the "UA says 120 but JS
// reports 148" inconsistency that some bot detectors flag. The value
// is intentionally hardcoded for now (a MYSCRAPER_USER_AGENT env
// var can be added later if the Vpass block changes shape). Sites
// that front themselves with Akamai Bot Manager or similar will
// still 403 the default Playwright fingerprint (TLS / HTTP/2
// signature, navigator.webdriver), so callers should also pass
// --disable-blink-features=AutomationControlled in
// MYSCRAPER_CHROMIUM_ARGS; that flag lives at the launch layer
// while this constant lives at the context layer.
const defaultUserAgent = "Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/148.0.0.0 Safari/537.36"

// withPage starts a Playwright-managed chromium browser, opens a new
// page, and hands it to run. It installs the driver, launches the
// browser, and tears everything down before returning so callers do
// not need to manage Playwright's lifecycle directly. When
// MYSCRAPER_DEBUG_DIR is set, the run also writes a JSONL trace of
// every HTTP response and workflow step into a file inside that
// directory.
func withPage(
	// ctx is part of the Browser interface contract and is reserved
	// for future cancellation; the current implementation does not
	// plumb it into the playwright driver.
	ctx context.Context,
	headless bool,
	run func(page playwright.Page, trace *debugTrace) (scrape.PageSnapshot, error),
) (scrape.PageSnapshot, error) {
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
		Args:           chromiumLaunchArgs(),
	})
	if err != nil {
		return scrape.PageSnapshot{}, fmt.Errorf("launch chromium: %w", err)
	}
	defer browser.Close()

	browserContext, err := browser.NewContext(playwright.BrowserNewContextOptions{
		UserAgent: playwright.String(defaultUserAgent),
	})
	if err != nil {
		return scrape.PageSnapshot{}, fmt.Errorf("new browser context: %w", err)
	}
	defer browserContext.Close()

	page, err := browserContext.NewPage()
	if err != nil {
		return scrape.PageSnapshot{}, fmt.Errorf("new page: %w", err)
	}

	trace := openDebugTrace()
	defer trace.close()
	if trace != nil {
		page.On("response", func(resp playwright.Response) {
			trace.recordResponse(resp)
		})
	}

	snapshot, runErr := run(page, trace)
	if runErr != nil && trace != nil {
		trace.recordError(runErr)
	}
	return snapshot, runErr
}

// Fetch opens url in a headless (or headed) Chromium driven by
// Playwright and returns the rendered page snapshot.
func (PlaywrightBrowser) Fetch(ctx context.Context, url string, headless bool) (scrape.PageSnapshot, error) {
	return withPage(ctx, headless, func(page playwright.Page, trace *debugTrace) (scrape.PageSnapshot, error) {
		if trace != nil {
			trace.recordStep("Fetch", url)
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
			URL:   page.URL(),
			Title: title,
			HTML:  html,
		}, nil
	})
}

// FetchSMBCCardWebMeisai runs the SMBC Card Vpass flow against the
// given credentials and returns the rendered WEB明細 page snapshot.
// The Playwright page is adapted to the smbccard.InteractivePage
// interface so the workflow itself stays in the smbccard package and
// can be exercised by unit tests with a fake page. The local
// smbccard.PageSnapshot is converted to scrape.PageSnapshot at the
// boundary to keep the smbccard package free of any scrape import.
func (PlaywrightBrowser) FetchSMBCCardWebMeisai(ctx context.Context, creds smbccard.Credentials, headless bool) (scrape.PageSnapshot, error) {
	return withPage(ctx, headless, func(page playwright.Page, trace *debugTrace) (scrape.PageSnapshot, error) {
		snapshot, err := smbccard.CaptureWebMeisai(playwrightInteractivePage{page: page, trace: trace}, creds)
		if err != nil {
			return scrape.PageSnapshot{}, err
		}
		return scrape.PageSnapshot{
			URL:   snapshot.URL,
			Title: snapshot.Title,
			HTML:  snapshot.HTML,
		}, nil
	})
}

// playwrightInteractivePage adapts a Playwright page to the
// smbccard.InteractivePage interface. It is unexported because the
// only legitimate consumer is FetchSMBCCardWebMeisai in this same
// package; callers outside the browser package should not need to
// reach for Playwright types. The trace field is optional; when
// nil, no step markers are written.
type playwrightInteractivePage struct {
	page  playwright.Page
	trace *debugTrace
}

func (p playwrightInteractivePage) Goto(url string) error {
	if p.trace != nil {
		p.trace.recordStep("Goto", url)
	}
	_, err := p.page.Goto(url, playwright.PageGotoOptions{
		WaitUntil: playwright.WaitUntilStateNetworkidle,
	})
	return err
}

func (p playwrightInteractivePage) FillByLabel(label, value string) error {
	// Per the InteractivePage contract, adapters must not include the
	// value (which can be a credential) in any returned error, and
	// must not include the value in the debug trace for the same
	// reason.
	if p.trace != nil {
		p.trace.recordStep("FillByLabel", label)
	}
	return p.page.GetByLabel(label).Fill(value)
}

func (p playwrightInteractivePage) ClickButton(name string) error {
	if p.trace != nil {
		p.trace.recordStep("ClickButton", name)
	}
	return p.page.GetByRole(*playwright.AriaRoleButton, playwright.PageGetByRoleOptions{
		Name: name,
	}).Click()
}

func (p playwrightInteractivePage) WaitForURL(pattern string) error {
	if p.trace != nil {
		p.trace.recordStep("WaitForURL", pattern)
	}
	return p.page.WaitForURL(pattern)
}

func (p playwrightInteractivePage) Title() (string, error) {
	return p.page.Title()
}

func (p playwrightInteractivePage) Content() (string, error) {
	return p.page.Content()
}

func (p playwrightInteractivePage) URL() string {
	return p.page.URL()
}
