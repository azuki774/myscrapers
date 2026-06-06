package moneyforward

import (
	"context"
	"time"
)

// Session is the minimum surface the MoneyForward orchestrators need
// from a real browser. The Playwright adapter (session_playwright.go)
// implements it for production; the fakeSession in fake.go implements
// it for unit and e2e tests (the latter via the NewFakeSession
// constructor).
type Session interface {
	AddCookies(ctx context.Context, cookies []Cookie) error
	Goto(ctx context.Context, url string) error
	Content(ctx context.Context) (string, error)
	ClickByXPath(ctx context.Context, xpath string) error
	ClickByText(ctx context.Context, text string) error
	ClickLinkIn(ctx context.Context, parentSelector, linkText string) error
	Wait(ctx context.Context, d time.Duration) error
	Close() error
}
