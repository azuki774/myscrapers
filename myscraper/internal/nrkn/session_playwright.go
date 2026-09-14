package nrkn

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"os"
	"strings"
	"time"

	browserpkg "github.com/azuki774/myscrapers/myscraper/internal/browser"
	"github.com/azuki774/myscrapers/myscraper/internal/chromium"
	"github.com/mxschmitt/playwright-go"
)

const nrknUserAgent = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/126.0.0.0 Safari/537.36"
const logoutSelector = "a[href*='W37S0020_Head.submit']:visible"

var installPlaywrightDriver = browserpkg.InstallDriver
var runPlaywright = playwright.Run
var _ Session = (*PlaywrightSession)(nil)

type PlaywrightSession struct {
	pw               *playwright.Playwright
	browser          playwright.Browser
	context          playwright.BrowserContext
	page             playwright.Page
	loginPageLoaded  bool
	loginSubmissions int
	concurrentLogin  bool
	logoutAttempted  bool
}

func NewPlaywrightSession(ctx context.Context, headless bool) (*PlaywrightSession, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if err := installPlaywrightDriver(); err != nil {
		return nil, errors.New("install playwright driver failed")
	}
	driverPath := os.Getenv("PLAYWRIGHT_DRIVER_PATH")
	if driverPath == "" {
		driverPath = ".playwright-driver"
	}
	pw, err := runPlaywright(&playwright.RunOptions{DriverDirectory: driverPath, SkipInstallBrowsers: true})
	if err != nil {
		return nil, errors.New("start playwright failed")
	}
	s := &PlaywrightSession{pw: pw}
	executable, err := chromium.ExecutablePath()
	if err != nil {
		_ = s.Close()
		return nil, err
	}
	s.browser, err = pw.Chromium.Launch(playwright.BrowserTypeLaunchOptions{
		ExecutablePath: playwright.String(executable), Headless: playwright.Bool(headless),
		Args: []string{"--no-sandbox", "--disable-dev-shm-usage", "--lang=ja-JP"},
	})
	if err != nil {
		_ = s.Close()
		return nil, errors.New("launch chromium failed")
	}
	s.context, err = s.browser.NewContext(playwright.BrowserNewContextOptions{
		Locale: playwright.String("ja-JP"), UserAgent: playwright.String(nrknUserAgent),
		Viewport: &playwright.Size{Width: 1440, Height: 1000},
	})
	if err != nil {
		_ = s.Close()
		return nil, errors.New("create browser context failed")
	}
	s.page, err = s.context.NewPage()
	if err != nil {
		_ = s.Close()
		return nil, errors.New("create page failed")
	}
	return s, nil
}

// Bound operations to the caller's deadline. Never wrap Playwright Fill errors:
// their call logs can contain the supplied credentials.
func (s *PlaywrightSession) budget(ctx context.Context) (float64, error) {
	if err := ctx.Err(); err != nil {
		return 0, err
	}
	d := pageTimeout
	if deadline, ok := ctx.Deadline(); ok && time.Until(deadline) < d {
		d = time.Until(deadline)
	}
	ms := float64(d.Milliseconds())
	if ms < 1 {
		return 0, context.DeadlineExceeded
	}
	s.page.SetDefaultTimeout(ms)
	s.page.SetDefaultNavigationTimeout(ms)
	return ms, nil
}
func (s *PlaywrightSession) pathIs(path string) bool {
	u, err := url.Parse(s.page.URL())
	return err == nil && u.Scheme == "https" && u.Host == "www.nrkn.co.jp" && u.Path == path
}
func (s *PlaywrightSession) clickNavigation(ctx context.Context, selector, operation string) error {
	ms, err := s.budget(ctx)
	if err != nil {
		return err
	}
	_, err = s.page.ExpectNavigation(func() error {
		return s.page.Locator(selector).First().Click(playwright.LocatorClickOptions{Timeout: playwright.Float(ms)})
	}, playwright.PageExpectNavigationOptions{WaitUntil: playwright.WaitUntilStateDomcontentloaded, Timeout: playwright.Float(ms)})
	if err != nil {
		return fmt.Errorf("nrkn: %s failed", operation)
	}
	return nil
}
func (s *PlaywrightSession) Login(ctx context.Context, id, password, birthday string) error {
	if s.loginSubmissions >= 2 || (s.loginSubmissions > 0 && !s.concurrentLogin) {
		return errors.New("nrkn: login retry limit reached")
	}
	if !s.loginPageLoaded {
		ms, err := s.budget(ctx)
		if err != nil {
			return err
		}
		if _, err = s.page.Goto(loginURL, playwright.PageGotoOptions{WaitUntil: playwright.WaitUntilStateLoad, Timeout: playwright.Float(ms)}); err != nil {
			return errors.New("nrkn: open login page failed")
		}
		s.loginPageLoaded = true
	}
	for _, field := range []struct{ name, value string }{{"userId", id}, {"password", password}, {"birthDate", birthday}} {
		if _, err := s.budget(ctx); err != nil {
			return err
		}
		if err := s.page.Locator("#pc_disp input[name='" + field.name + "']:visible").Fill(field.value); err != nil {
			return errors.New("nrkn: fill login field failed")
		}
	}
	s.loginSubmissions++
	s.concurrentLogin = false
	if err := s.clickNavigation(ctx, "#pc_disp #btnLogin:visible", "submit login"); err != nil {
		return err
	}
	// NRKN may render an intermediate document before navigating to its menu.
	// Waiting observes that navigation; it never submits credentials again.
	initialText, _ := s.page.Locator("body").InnerText()
	if strings.Contains(initialText, "990003") {
		s.concurrentLogin = true
		return ErrConcurrentLogin
	}
	if visible, _ := s.page.Locator("#pc_disp #btnLogin").IsVisible(); visible {
		return errors.New("nrkn: authentication failed or service unavailable")
	}
	ms, err := s.budget(ctx)
	if err != nil {
		return err
	}
	_ = s.page.WaitForURL("https://www.nrkn.co.jp/webapp/nrk/W37S0030_View.do", playwright.PageWaitForURLOptions{WaitUntil: playwright.WaitUntilStateLoad, Timeout: playwright.Float(ms)})
	if s.pathIs("/webapp/nrk/W37S0030_View.do") {
		if _, err := s.budget(ctx); err != nil {
			return err
		}
		if err := s.page.Locator(logoutSelector).First().WaitFor(); err != nil {
			return errors.New("nrkn: authenticated menu not found")
		}
		return nil
	}
	text, err := s.page.Locator("body").InnerText()
	if err == nil && strings.Contains(text, "990003") {
		s.concurrentLogin = true
		return ErrConcurrentLogin
	}
	return errors.New("nrkn: authentication failed or service unavailable")
}
func (s *PlaywrightSession) dismissNotice(ctx context.Context) error {
	if _, err := s.budget(ctx); err != nil {
		return err
	}
	visible, err := s.page.Locator("#myDialog[open]").IsVisible()
	if err != nil {
		return errors.New("nrkn: inspect notice failed")
	}
	if visible {
		if err := s.page.Locator("#myDialog #btnClose:visible").Click(); err != nil {
			return errors.New("nrkn: close notice failed")
		}
		if err := s.page.Locator("#myDialog").WaitFor(playwright.LocatorWaitForOptions{State: playwright.WaitForSelectorStateHidden}); err != nil {
			return errors.New("nrkn: notice remained open")
		}
	}
	return nil
}
func (s *PlaywrightSession) NavigateToAssets(ctx context.Context) error {
	if err := s.dismissNotice(ctx); err != nil {
		return err
	}
	if err := s.clickNavigation(ctx, "a[href*='W37S1040_Form.submit']:visible", "open asset valuation"); err != nil {
		return err
	}
	if !s.pathIs("/webapp/nrk/W37S1040_AssetValuePlan.do") {
		return errors.New("nrkn: unexpected asset valuation destination")
	}
	if _, err := s.budget(ctx); err != nil {
		return err
	}
	if err := s.page.Locator("#W37S104001").WaitFor(); err != nil {
		return errors.New("nrkn: asset valuation content not ready")
	}
	return nil
}
func (s *PlaywrightSession) BodyHTML(ctx context.Context) (string, error) {
	if _, err := s.budget(ctx); err != nil {
		return "", err
	}
	h, err := s.page.Content()
	if err != nil {
		return "", errors.New("nrkn: read asset page failed")
	}
	return h, nil
}
func (s *PlaywrightSession) Logout(ctx context.Context) error {
	if s.logoutAttempted {
		return errors.New("nrkn: logout already attempted")
	}
	if s.page == nil {
		return nil
	}
	if err := s.dismissNotice(ctx); err != nil {
		return err
	}
	n, err := s.page.Locator(logoutSelector).Count()
	if err != nil {
		return errors.New("nrkn: inspect logout link failed")
	}
	if n == 0 {
		if visible, _ := s.page.Locator("#pc_disp #btnLogin").IsVisible(); visible {
			return nil
		}
		return errors.New("nrkn: logout link unavailable; session end unconfirmed")
	}
	s.logoutAttempted = true
	if err := s.clickNavigation(ctx, logoutSelector, "logout"); err != nil {
		return err
	}
	n, err = s.page.Locator(logoutSelector).Count()
	if err != nil || n != 0 {
		return errors.New("nrkn: logout completion not confirmed")
	}
	if visible, _ := s.page.Locator("#pc_disp #btnLogin").IsVisible(); visible {
		return nil
	}
	text, err := s.page.Locator("body").InnerText()
	if err == nil && s.pathIs("/webapp/nrk/W37S0020_View.do") {
		for _, phrase := range []string{"ログアウトしました", "ログアウトされました", "終了しました", "ご利用ありがとうございました", "ご利用いただきありがとうございました"} {
			if strings.Contains(text, phrase) {
				return nil
			}
		}
	}
	return errors.New("nrkn: logout completion not confirmed")
}
func (s *PlaywrightSession) Close() error {
	var errs []error
	if s.context != nil {
		errs = append(errs, s.context.Close())
		s.context = nil
	}
	if s.browser != nil {
		errs = append(errs, s.browser.Close())
		s.browser = nil
	}
	if s.pw != nil {
		errs = append(errs, s.pw.Stop())
		s.pw = nil
	}
	return errors.Join(errs...)
}
