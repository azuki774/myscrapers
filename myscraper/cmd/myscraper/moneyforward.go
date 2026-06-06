package main

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/azuki774/myscrapers/myscraper/internal/moneyforward"
)

// moneyforwardRunner is the production cli.MoneyforwardRunner: it opens
// a Playwright-backed Session, optionally builds an S3Uploader, and
// delegates to moneyforward.Fetch / moneyforward.Update.
type moneyforwardRunner struct {
	logger   *slog.Logger
	headless bool
}

func (r moneyforwardRunner) RunFetch(ctx context.Context, opts moneyforward.FetchOptions, s3Upload bool) error {
	sess, err := moneyforward.NewPlaywrightSession(ctx, r.headless)
	if err != nil {
		return fmt.Errorf("open session: %w", err)
	}
	opts.Session = sess
	if s3Upload {
		upl, err := moneyforward.NewS3Uploader(ctx)
		if err != nil {
			return fmt.Errorf("build uploader: %w", err)
		}
		opts.Uploader = upl
	}
	return moneyforward.Fetch(ctx, opts)
}

func (r moneyforwardRunner) RunUpdate(ctx context.Context, opts moneyforward.UpdateOptions) error {
	sess, err := moneyforward.NewPlaywrightSession(ctx, r.headless)
	if err != nil {
		return fmt.Errorf("open session: %w", err)
	}
	opts.Session = sess
	return moneyforward.Update(ctx, opts)
}
