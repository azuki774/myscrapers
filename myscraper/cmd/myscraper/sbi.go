package main

import (
	"context"
	"fmt"
	"io"
	"log/slog"

	"github.com/azuki774/myscrapers/myscraper/internal/cli"
	"github.com/azuki774/myscrapers/myscraper/internal/sbi"
)

// sbiRunner is the production cli.SBIRunner: it opens a
// Playwright-backed Session, logs in with the saved passkey, fetches
// the asset summary, and writes JSON.
type sbiRunner struct {
	logger *slog.Logger
	stdout io.Writer
}

func (r sbiRunner) RunAssets(ctx context.Context, opts sbi.FetchOptions) error {
	r.logger.Info("opening Playwright session", "headless", opts.Headless)
	sess, err := sbi.NewPlaywrightSession(ctx, opts.Headless)
	if err != nil {
		return fmt.Errorf("open session: %w", err)
	}
	defer func() {
		if err := sess.Close(); err != nil {
			r.logger.Warn("closing session", "error", err)
		}
	}()
	r.logger.Info("logging in with passkey", "path", opts.PasskeyPath)
	if err := sess.LoginWithPasskey(ctx, opts.Passkey); err != nil {
		return fmt.Errorf("passkey login: %w", err)
	}
	r.logger.Info("login succeeded, fetching assets")
	assets, err := sbi.FetchAssets(ctx, sess, opts.Now)
	if err != nil {
		return fmt.Errorf("fetch assets: %w", err)
	}
	if err := cli.WriteAssetsJSON(r.stdout, opts.OutputPath, assets); err != nil {
		return fmt.Errorf("write assets json: %w", err)
	}
	r.logger.Info("assets fetched", "grand_total_jpy", assets.GrandTotalJPY)
	return nil
}
