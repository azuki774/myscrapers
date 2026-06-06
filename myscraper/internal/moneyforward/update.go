package moneyforward

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"time"
)

const (
	accountsURL     = "https://moneyforward.com/accounts"
	homeURL         = "https://moneyforward.com/"
	bulkUpdateXPath = "/html/body/div[1]/div[2]/div[1]/div/div/div/section/p[2]/a"
	accountListSel  = "li.account.facilities-column"
	suicaUpdateText = "更新"
	postUpdateSleep = 60 * time.Second
	postHomeSleep   = 10 * time.Second
)

// UpdateOptions mirrors FetchOptions for the Update subcommand. SuicaPage
// is optional and currently unused at the Session level — the Suica
// search happens via ClickLinkIn which the fakeSession asserts.
type UpdateOptions struct {
	Session Session
	Cookies []Cookie
	Logger  *slog.Logger
}

// Update orchestrates the "moneyforward --update" subcommand: prime
// cookies, click the bulk update button, then locate the Suica account
// in the home page account list and click its "更新" link. Returns nil
// when the Suica link is not found (matching the Python "log warning
// and continue" behaviour).
func Update(ctx context.Context, opts UpdateOptions) error {
	if opts.Session == nil {
		return fmt.Errorf("session is required")
	}
	logger := opts.Logger
	if logger == nil {
		logger = slog.New(slog.NewJSONHandler(io.Discard, nil))
	}
	defer opts.Session.Close()

	if err := opts.Session.Goto(ctx, cfURL); err != nil {
		return fmt.Errorf("goto cf: %w", err)
	}
	if err := opts.Session.Wait(ctx, cookiePrimSleep); err != nil {
		return fmt.Errorf("wait after cf: %w", err)
	}
	if err := opts.Session.AddCookies(ctx, opts.Cookies); err != nil {
		return fmt.Errorf("add cookies: %w", err)
	}

	if err := opts.Session.Goto(ctx, accountsURL); err != nil {
		return fmt.Errorf("goto accounts: %w", err)
	}
	if err := opts.Session.ClickByXPath(ctx, bulkUpdateXPath); err != nil {
		return fmt.Errorf("click bulk update: %w", err)
	}
	if err := opts.Session.Wait(ctx, postUpdateSleep); err != nil {
		return fmt.Errorf("wait after bulk update: %w", err)
	}
	logger.Info("pressed bulk update button")

	if err := opts.Session.Goto(ctx, homeURL); err != nil {
		return fmt.Errorf("goto home: %w", err)
	}
	if err := opts.Session.Wait(ctx, postHomeSleep); err != nil {
		return fmt.Errorf("wait after home: %w", err)
	}
	if err := opts.Session.ClickLinkIn(ctx, accountListSel, suicaUpdateText); err != nil {
		logger.Warn("could not find Suica account or 更新 link; skipping", "error", err)
		return nil
	}
	if err := opts.Session.Wait(ctx, postUpdateSleep); err != nil {
		return fmt.Errorf("wait after suica update: %w", err)
	}
	logger.Info("pressed Suica 更新 link")
	return nil
}
