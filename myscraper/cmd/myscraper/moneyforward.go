package main

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/azuki774/myscrapers/myscraper/internal/moneyforward"
)

// moneyforwardRunner is the production cli.MoneyforwardRunner: it opens
// a Playwright-backed Session, optionally builds an S3Store, and
// delegates to moneyforward.Fetch / moneyforward.Update.
type moneyforwardRunner struct {
	logger   *slog.Logger
	headless bool
}

func (r moneyforwardRunner) RunFetch(ctx context.Context, opts moneyforward.FetchOptions, s3Upload bool) error {
	r.logger.Info("opening Playwright session", "headless", r.headless)
	sess, err := moneyforward.NewPlaywrightSession(ctx, r.headless)
	if err != nil {
		return fmt.Errorf("open session: %w", err)
	}
	r.logger.Info("Playwright session opened")
	opts.Session = sess
	if s3Upload {
		r.logger.Info("creating S3 uploader")
		upl, err := moneyforward.NewS3Store(ctx)
		if err != nil {
			return fmt.Errorf("build uploader: %w", err)
		}
		opts.Uploader = upl
		r.logger.Info("S3 uploader created")
	}
	r.logger.Info("starting fetch")
	return moneyforward.Fetch(ctx, opts)
}

func (r moneyforwardRunner) RunUpdate(ctx context.Context, opts moneyforward.UpdateOptions) error {
	r.logger.Info("opening Playwright session", "headless", r.headless)
	sess, err := moneyforward.NewPlaywrightSession(ctx, r.headless)
	if err != nil {
		return fmt.Errorf("open session: %w", err)
	}
	r.logger.Info("Playwright session opened")
	opts.Session = sess
	r.logger.Info("starting update")
	return moneyforward.Update(ctx, opts)
}

func (r moneyforwardRunner) RunFetchCookie(ctx context.Context, cookiePath string) error {
	return fmt.Errorf("fetch-cookie not yet implemented")
}
