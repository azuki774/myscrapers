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
	"time"

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

// hardStepTimeout caps each individual InteractivePage operation
// (Goto, Fill, Click, WaitForURL). The page-level Timeout option
// in playwright only covers the navigation state, not the
// underlying HTTP request, and the Go binding's channel.Send blocks
// until the browser replies. A server that never even sends
// headers would otherwise leave the operation hanging forever.
const hardStepTimeout = 20 * time.Second

// hardRunTimeout caps the entire browser run (the run callback
// passed to withPage). Even if each step has its own hard timeout,
// we still want an outer cap so a series of failed steps cannot
// accumulate to a multi-minute hang.
const hardRunTimeout = 90 * time.Second

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
	// Hard outer cap on the entire run. The page-level Timeout
	// option only covers the navigation state, not the underlying
	// HTTP request, and the Go binding's channel.Send blocks until
	// the browser replies. If Vpass (or any other site) hangs the
	// TCP connection, the navigation never starts and the page
	// timeout never fires. The outer cap aborts the workflow and
	// returns a clear error so the user is not stuck ^C-ing the
	// process. The browser and driver are torn down by the
	// deferred Close / Stop calls regardless of which path returns.
	ctx, cancel := context.WithTimeout(ctx, hardRunTimeout)
	defer cancel()

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

	type runResult struct {
		snap scrape.PageSnapshot
		err  error
	}
	done := make(chan runResult, 1)
	go func() {
		s, e := run(page, trace)
		done <- runResult{s, e}
	}()

	select {
	case r := <-done:
		if r.err != nil && trace != nil {
			trace.recordError(r.err)
		}
		return r.snap, r.err
	case <-ctx.Done():
		err := fmt.Errorf("browser run exceeded hard timeout of %s", hardRunTimeout)
		if trace != nil {
			trace.recordError(err)
		}
		return scrape.PageSnapshot{}, err
	}
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
	// WaitUntilStateCommit is the most lenient wait condition: it
	// returns as soon as the browser has received the response
	// headers. Vpass keeps connections open with analytics and
	// long-polling, so the stricter conditions (load /
	// domcontentloaded / networkidle) may never fire, and the page
	// timeout option only covers the navigation state, not the
	// underlying HTTP request. commit is also wrapped in a
	// goroutine with a hard timeout, so a server that never even
	// sends headers can no longer hang the workflow.
	done := make(chan error, 1)
	go func() {
		_, err := p.page.Goto(url, playwright.PageGotoOptions{
			WaitUntil: playwright.WaitUntilStateCommit,
			Timeout:   playwright.Float(15000),
		})
		done <- err
	}()
	select {
	case err := <-done:
		return err
	case <-time.After(hardStepTimeout):
		return fmt.Errorf("goto %s: hard timeout of %s exceeded (page-level Goto did not return)", url, hardStepTimeout)
	}
}

func (p playwrightInteractivePage) FillByLabel(label, value string) error {
	// Per the InteractivePage contract, adapters must not include the
	// value (which can be a credential) in any returned error, and
	// must not include the value in the debug trace for the same
	// reason.
	if p.trace != nil {
		p.trace.recordStep("FillByLabel", label)
	}
	done := make(chan error, 1)
	go func() {
		done <- p.page.GetByLabel(label).Fill(value)
	}()
	select {
	case err := <-done:
		return err
	case <-time.After(hardStepTimeout):
		return fmt.Errorf("fill %s: hard timeout of %s exceeded", label, hardStepTimeout)
	}
}

func (p playwrightInteractivePage) ClickButton(name string) error {
	if p.trace != nil {
		p.trace.recordStep("ClickButton", name)
	}
	done := make(chan error, 1)
	go func() {
		done <- p.page.GetByRole(*playwright.AriaRoleButton, playwright.PageGetByRoleOptions{
			Name: name,
		}).Click()
	}()
	select {
	case err := <-done:
		return err
	case <-time.After(hardStepTimeout):
		return fmt.Errorf("click %s: hard timeout of %s exceeded", name, hardStepTimeout)
	}
}

func (p playwrightInteractivePage) WaitForURL(pattern string) error {
	if p.trace != nil {
		p.trace.recordStep("WaitForURL", pattern)
	}
	// Mirror the Goto hard-timeout pattern. WaitForURL has a
	// 15s CDP timeout, but the channel.Send call can still hang
	// indefinitely when the browser is unresponsive.
	done := make(chan error, 1)
	go func() {
		done <- p.page.WaitForURL(pattern, playwright.PageWaitForURLOptions{
			Timeout: playwright.Float(15000),
		})
	}()
	select {
	case err := <-done:
		return err
	case <-time.After(hardStepTimeout):
		return fmt.Errorf("wait for url %s: hard timeout of %s exceeded", pattern, hardStepTimeout)
	}
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
