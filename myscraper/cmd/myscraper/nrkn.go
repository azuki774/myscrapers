package main

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"time"

	"github.com/azuki774/myscrapers/myscraper/internal/cli"
	"github.com/azuki774/myscrapers/myscraper/internal/nrkn"
)

type nrknRunner struct {
	logger *slog.Logger
	stdout io.Writer
}

func (r nrknRunner) RunAssets(ctx context.Context, opts nrkn.FetchOptions) error {
	s, err := nrkn.NewPlaywrightSession(ctx, opts.Headless)
	if err != nil {
		return fmt.Errorf("open session: %w", err)
	}
	defer func() {
		if err := s.Close(); err != nil {
			r.logger.Warn("closing session", "error", err)
		}
	}()
	fetchCtx, cancel := context.WithTimeout(ctx, 3*time.Minute)
	defer cancel()
	if opts.FIGIResolver == nil {
		opts.FIGIResolver = nrkn.NewHoldingFIGIResolver(nil)
	}
	a, fetchErr := nrkn.FetchAssets(fetchCtx, s, opts)
	if a == nil {
		return fetchErr
	}
	raw, err := cli.WriteNRKNJSON(r.stdout, opts.OutputPath, a)
	if err != nil {
		return fmt.Errorf("write assets json: %w", err)
	}
	if opts.S3Upload {
		key, err := cli.UploadNRKNJSON(ctx, opts.S3Client, opts.Now, raw)
		if err != nil {
			return fmt.Errorf("upload assets json: %w", err)
		}
		r.logger.Info("uploaded assets JSON", "key", key)
	}
	r.logger.Info("assets fetched", "grand_total_jpy", a.GrandTotalJPY)
	return fetchErr
}
