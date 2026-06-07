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

	logger.Info("navigating to CF page for cookie priming")
	if err := opts.Session.Goto(ctx, cfURL); err != nil {
		return fmt.Errorf("goto cf: %w", err)
	}
	logger.Info("waiting for cookie prim page to load", "duration", cookiePrimSleep)
	if err := opts.Session.Wait(ctx, cookiePrimSleep); err != nil {
		return fmt.Errorf("wait after cf: %w", err)
	}
	logger.Info("injecting cookies", "count", len(opts.Cookies))
	if err := opts.Session.AddCookies(ctx, opts.Cookies); err != nil {
		return fmt.Errorf("add cookies: %w", err)
	}

	logger.Info("navigating to accounts page")
	if err := opts.Session.Goto(ctx, accountsURL); err != nil {
		return fmt.Errorf("goto accounts: %w", err)
	}
	logger.Info("clicking bulk update button")
	if err := opts.Session.ClickByXPath(ctx, bulkUpdateXPath); err != nil {
		return fmt.Errorf("click bulk update: %w", err)
	}
	logger.Info("waiting for bulk update to complete", "duration", postUpdateSleep)
	if err := opts.Session.Wait(ctx, postUpdateSleep); err != nil {
		return fmt.Errorf("wait after bulk update: %w", err)
	}
	logger.Info("pressed bulk update button")

	logger.Info("navigating to home page for Suica update")
	if err := opts.Session.Goto(ctx, homeURL); err != nil {
		return fmt.Errorf("goto home: %w", err)
	}
	logger.Info("waiting for home page", "duration", postHomeSleep)
	if err := opts.Session.Wait(ctx, postHomeSleep); err != nil {
		return fmt.Errorf("wait after home: %w", err)
	}
	logger.Info("looking for Suica 更新 link")
	if err := opts.Session.ClickLinkIn(ctx, accountListSel, suicaUpdateText); err != nil {
		logger.Warn("could not find Suica account or 更新 link; skipping", "error", err)
		return nil
	}
	logger.Info("waiting for Suica update to complete", "duration", postUpdateSleep)
	if err := opts.Session.Wait(ctx, postUpdateSleep); err != nil {
		return fmt.Errorf("wait after suica update: %w", err)
	}
	logger.Info("pressed Suica 更新 link")
	return nil
}
