package smbccard

import (
	"fmt"

	"github.com/azuki774/myscrapers/myscraper/internal/scrape"
)

// Top of the SMBC Card public site. The login flow expects some
// session state from this page, so the workflow visits it before
// opening the login form.
const (
	TopURL              = "https://www.smbc-card.com/index.jsp"
	LoginURL            = "https://www.smbc-card.com/mem/index.jsp"
	MyPageURL           = "https://www.smbc-card.com/memx/mypage/index.html"
	MyPageURLPattern    = "**/memx/mypage/index.html"
	WebMeisaiURL        = "https://www.smbc-card.com/memx/web_meisai/top/index.html#info2"
	WebMeisaiURLPattern = "**/memx/web_meisai/top/index.html*"
)

// InteractivePage is the minimum browser surface area the SMBC
// workflow needs. The Playwright adapter (browser package) and the
// fakePage in unit tests both satisfy it, so the workflow stays
// browser-agnostic and easy to drive from tests.
// Adapters MUST NOT include the value passed to FillByLabel in any
// returned error, since the value can be a credential and the
// workflow wraps those errors and surfaces them to the caller.
type InteractivePage interface {
	Goto(url string) error
	FillByLabel(label, value string) error
	ClickButton(name string) error
	WaitForURL(pattern string) error
	Title() (string, error)
	Content() (string, error)
	URL() string
}

// CaptureWebMeisai drives the SMBC Card member site end-to-end: open
// the top page, open the login form, fill it in, submit, wait until
// the post-login mypage is reached, then navigate to the WEB明細
// page and wait for it to settle. It returns the rendered page
// snapshot of the WEB明細 page; the caller is responsible for
// persisting it.
func CaptureWebMeisai(page InteractivePage, creds Credentials) (scrape.PageSnapshot, error) {
	if err := page.Goto(TopURL); err != nil {
		return scrape.PageSnapshot{}, fmt.Errorf("goto top page: %w", err)
	}
	if err := page.Goto(LoginURL); err != nil {
		return scrape.PageSnapshot{}, fmt.Errorf("goto login page: %w", err)
	}
	if err := page.FillByLabel("VpassID", creds.LoginID); err != nil {
		return scrape.PageSnapshot{}, fmt.Errorf("fill VpassID: %w", err)
	}
	if err := page.FillByLabel("パスワード", creds.Password); err != nil {
		return scrape.PageSnapshot{}, fmt.Errorf("fill password: %w", err)
	}
	if err := page.ClickButton("ログイン"); err != nil {
		return scrape.PageSnapshot{}, fmt.Errorf("click login: %w", err)
	}
	if err := page.WaitForURL(MyPageURLPattern); err != nil {
		return scrape.PageSnapshot{}, fmt.Errorf("wait for mypage: %w", err)
	}
	if err := page.Goto(WebMeisaiURL); err != nil {
		return scrape.PageSnapshot{}, fmt.Errorf("goto web meisai page: %w", err)
	}
	if err := page.WaitForURL(WebMeisaiURLPattern); err != nil {
		return scrape.PageSnapshot{}, fmt.Errorf("wait for web meisai page: %w", err)
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
}
