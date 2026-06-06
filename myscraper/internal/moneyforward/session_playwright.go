package moneyforward

import (
	"context"
	"fmt"
	"time"

	"github.com/azuki774/myscrapers/myscraper/internal/chromium"
	"github.com/playwright-community/playwright-go"
)

// Compile-time check that PlaywrightSession satisfies Session.
var _ Session = (*PlaywrightSession)(nil)

// PlaywrightSession is the production implementation of Session backed
// by a single Playwright runtime, browser, context, and page. The page
// is reused across calls so cookies persist between navigations.
type PlaywrightSession struct {
	pw      *playwright.Playwright
	browser playwright.Browser
	context playwright.BrowserContext
	page    playwright.Page
}

// NewPlaywrightSession starts Playwright, launches a headless chromium
// driven by the system browser (chromium in the dev shell, google-chrome
// in the Docker image), opens a fresh BrowserContext, and returns a
// Session ready for the orchestrators to drive.
func NewPlaywrightSession(ctx context.Context, headless bool) (*PlaywrightSession, error) {
	pw, err := playwright.Run()
	if err != nil {
		return nil, fmt.Errorf("start playwright: %w", err)
	}
	executablePath, err := chromium.ExecutablePath()
	if err != nil {
		_ = pw.Stop()
		return nil, err
	}
	browser, err := pw.Chromium.Launch(playwright.BrowserTypeLaunchOptions{
		ExecutablePath: playwright.String(executablePath),
		Headless:       playwright.Bool(headless),
	})
	if err != nil {
		_ = pw.Stop()
		return nil, fmt.Errorf("launch chromium: %w", err)
	}
	bctx, err := browser.NewContext()
	if err != nil {
		_ = browser.Close()
		_ = pw.Stop()
		return nil, fmt.Errorf("new context: %w", err)
	}
	page, err := bctx.NewPage()
	if err != nil {
		_ = bctx.Close()
		_ = browser.Close()
		_ = pw.Stop()
		return nil, fmt.Errorf("new page: %w", err)
	}
	return &PlaywrightSession{pw: pw, browser: browser, context: bctx, page: page}, nil
}

func (s *PlaywrightSession) AddCookies(_ context.Context, cookies []Cookie) error {
	pcs := make([]playwright.OptionalCookie, 0, len(cookies))
	for _, c := range cookies {
		pc := playwright.OptionalCookie{
			Name:     c.Name,
			Value:    c.Value,
			Domain:   playwright.String(c.Domain),
			Path:     playwright.String(c.Path),
			Secure:   playwright.Bool(c.Secure),
			HttpOnly: playwright.Bool(c.HTTPOnly),
		}
		if c.Expires > 0 {
			pc.Expires = playwright.Float(c.Expires)
		}
		if c.SameSite != "" {
			ss := playwright.SameSiteAttribute(c.SameSite)
			pc.SameSite = &ss
		}
		pcs = append(pcs, pc)
	}
	return s.context.AddCookies(pcs)
}

func (s *PlaywrightSession) Goto(_ context.Context, url string) error {
	if _, err := s.page.Goto(url); err != nil {
		return fmt.Errorf("goto %s: %w", url, err)
	}
	return nil
}

func (s *PlaywrightSession) Content(_ context.Context) (string, error) {
	html, err := s.page.Content()
	if err != nil {
		return "", fmt.Errorf("read content: %w", err)
	}
	return html, nil
}

func (s *PlaywrightSession) ClickByXPath(_ context.Context, xpath string) error {
	loc := s.page.Locator("xpath=" + xpath)
	if err := loc.Click(); err != nil {
		return fmt.Errorf("click xpath %s: %w", xpath, err)
	}
	return nil
}

func (s *PlaywrightSession) ClickByText(_ context.Context, text string) error {
	loc := s.page.GetByText(text)
	if err := loc.Click(); err != nil {
		return fmt.Errorf("click text %q: %w", text, err)
	}
	return nil
}

func (s *PlaywrightSession) ClickLinkIn(_ context.Context, parentSelector, linkText string) error {
	loc := s.page.Locator(parentSelector).GetByText(linkText)
	if err := loc.Click(); err != nil {
		return fmt.Errorf("click %q in %q: %w", linkText, parentSelector, err)
	}
	return nil
}

func (s *PlaywrightSession) Wait(_ context.Context, d time.Duration) error {
	s.page.WaitForTimeout(float64(d.Milliseconds()))
	return nil
}

// Close tears down the page, context, browser, and Playwright driver
// in that order, returning the first error encountered (subsequent
// errors are swallowed because there is nothing useful to do with
// them). Each resource is nil-ed out after a close attempt so a second
// Close call is a no-op returning nil, which lets callers chain a
// defer-Close with an early-error Close without seeing spurious errors.
func (s *PlaywrightSession) Close() error {
	var firstErr error
	if s.context != nil {
		if err := s.context.Close(); err != nil {
			firstErr = fmt.Errorf("close context: %w", err)
		}
		s.context = nil
	}
	if s.browser != nil {
		if err := s.browser.Close(); err != nil && firstErr == nil {
			firstErr = fmt.Errorf("close browser: %w", err)
		}
		s.browser = nil
	}
	if s.pw != nil {
		if err := s.pw.Stop(); err != nil && firstErr == nil {
			firstErr = fmt.Errorf("stop playwright: %w", err)
		}
		s.pw = nil
	}
	return firstErr
}
