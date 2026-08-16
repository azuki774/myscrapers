package sbi

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	browserpkg "github.com/azuki774/myscrapers/myscraper/internal/browser"
	"github.com/azuki774/myscrapers/myscraper/internal/chromium"
	"github.com/mxschmitt/playwright-go"
)

// ErrLoginFailed reports that the restored passkey did not result in an
// authenticated ETGate session.
var ErrLoginFailed = errors.New("sbi: passkey login failed")

// sbiUserAgent is a normal desktop Chrome UA. SBI's login API rejects
// HeadlessChrome user agents (302 to a maintenance page), so the UA must
// not expose headless automation.
const sbiUserAgent = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/126.0.0.0 Safari/537.36"

// loginURL is the entry point that performs the passkey ceremony.
const loginURL = "https://login.sbisec.co.jp/login/entry"

// portalURLPrefix is the authenticated destination after a successful
// passkey login.
const portalURLPrefix = "https://site1.sbisec.co.jp/ETGate/"

var (
	defaultInstallPlaywrightDriver = browserpkg.InstallDriver
	installPlaywrightDriver        = defaultInstallPlaywrightDriver
	defaultRunPlaywright           = playwright.Run
	runPlaywright                  = defaultRunPlaywright
)

// Compile-time check that PlaywrightSession satisfies Session.
var _ Session = (*PlaywrightSession)(nil)

// PlaywrightSession is the production Session backed by a single
// Playwright runtime, browser, context, and page.
type PlaywrightSession struct {
	pw      *playwright.Playwright
	browser playwright.Browser
	context playwright.BrowserContext
	page    playwright.Page
}

// NewPlaywrightSession starts Playwright, launches headless chromium
// with the SBI user agent, and returns a Session.
func NewPlaywrightSession(ctx context.Context, headless bool) (*PlaywrightSession, error) {
	if err := installPlaywrightDriver(); err != nil {
		return nil, fmt.Errorf("install playwright driver: %w", err)
	}
	pw, err := runPlaywright()
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
		Args: []string{
			"--no-sandbox",
			"--disable-gpu",
			"--lang=ja-JP",
			"--disable-dev-shm-usage",
		},
	})
	if err != nil {
		_ = pw.Stop()
		return nil, fmt.Errorf("launch chromium: %w", err)
	}
	bctx, err := browser.NewContext(playwright.BrowserNewContextOptions{
		UserAgent: playwright.String(sbiUserAgent),
		Locale:    playwright.String("ja-JP"),
	})
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

// LoginWithPasskey installs the saved credentials into a WebAuthn
// virtual authenticator via the Chrome DevTools Protocol, then
// navigates to the SBI login URL. With a discoverable credential
// present, SBI signs in automatically and redirects to the ETGate
// portal; when that does not happen within the timeout the passkey
// button is clicked once more before giving up.
func (s *PlaywrightSession) LoginWithPasskey(ctx context.Context, passkey *PasskeyFile) error {
	cdp, err := s.context.NewCDPSession(s.page)
	if err != nil {
		return fmt.Errorf("open cdp session: %w", err)
	}
	if _, err := cdp.Send("WebAuthn.enable", map[string]interface{}{"enableDiscovery": true}); err != nil {
		return fmt.Errorf("enable webauthn: %w", err)
	}
	created, err := cdp.Send("WebAuthn.addVirtualAuthenticator", map[string]interface{}{
		"options": map[string]interface{}{
			"protocol":             "ctap2",
			"transport":            "internal",
			"hasResidentKey":       true,
			"hasUserVerification":  true,
			"isUserVerified":       true,
		},
	})
	if err != nil {
		return fmt.Errorf("add virtual authenticator: %w", err)
	}
	authID, ok := created.(map[string]interface{})["authenticatorId"].(string)
	if !ok || authID == "" {
		return fmt.Errorf("virtual authenticator returned no id")
	}
	for _, c := range passkey.Credentials {
		if _, err := cdp.Send("WebAuthn.addCredential", map[string]interface{}{
			"authenticatorId": authID,
			"credential": map[string]interface{}{
				"credentialId":         c.CredentialID,
				"isResidentCredential": c.IsResidentCredential,
				"rpId":                 c.RPID,
				"privateKey":           c.PrivateKey,
				"userHandle":           c.UserHandle,
				"signCount":            c.SignCount,
			},
		}); err != nil {
			return fmt.Errorf("add credential: %w", err)
		}
	}

	if err := s.Goto(ctx, loginURL); err != nil {
		return err
	}
	for attempt := 0; attempt < 3; attempt++ {
		if err := s.waitForPortal(ctx); err == nil {
			return nil
		}
		btn := s.page.GetByRole("button", playwright.PageGetByRoleOptions{Name: "パスキー認証でログイン"})
		if count, err := btn.Count(); err == nil && count > 0 {
			if err := btn.Click(); err == nil {
				time.Sleep(2 * time.Second)
				continue
			}
		}
		break
	}
	return ErrLoginFailed
}

// waitForPortal polls the current URL until the authenticated ETGate
// portal is reached.
func (s *PlaywrightSession) waitForPortal(ctx context.Context) error {
	deadline := time.Now().Add(20 * time.Second)
	for time.Now().Before(deadline) {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}
		url := s.page.URL()
		if strings.HasPrefix(url, portalURLPrefix) {
			return nil
		}
		time.Sleep(500 * time.Millisecond)
	}
	return fmt.Errorf("portal not reached, url=%s", s.page.URL())
}

func (s *PlaywrightSession) Goto(_ context.Context, url string) error {
	if _, err := s.page.Goto(url, playwright.PageGotoOptions{
		WaitUntil: playwright.WaitUntilStateDomcontentloaded,
	}); err != nil {
		return fmt.Errorf("goto %s: %w", url, err)
	}
	return nil
}

func (s *PlaywrightSession) BodyText(_ context.Context) (string, error) {
	text, err := s.page.Locator("body").InnerText()
	if err != nil {
		return "", fmt.Errorf("read body text: %w", err)
	}
	return text, nil
}

func (s *PlaywrightSession) Wait(_ context.Context, d time.Duration) error {
	s.page.WaitForTimeout(float64(d.Milliseconds()))
	return nil
}

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
