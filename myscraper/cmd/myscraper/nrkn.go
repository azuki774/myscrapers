package main

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"time"

	"github.com/azuki774/myscrapers/myscraper/internal/cli"
	"github.com/azuki774/myscrapers/myscraper/internal/logging"
	"github.com/azuki774/myscrapers/myscraper/internal/nrkn"
)

type nrknRunner struct {
	logger *slog.Logger
	stdout io.Writer
}

var newNRKNSession = func(ctx context.Context, headless bool) (nrkn.Session, error) {
	return nrkn.NewPlaywrightSession(ctx, headless)
}
var newNRKNFIGIResolver = func() nrkn.HoldingFIGIResolver {
	return nrkn.NewHoldingFIGIResolver(nil)
}

func (r nrknRunner) RunAssets(ctx context.Context, opts nrkn.FetchOptions) error {
	log := logging.New(r.logger, "nrkn")
	sessionStarted := log.Started("session", "headless", opts.Headless)
	s, err := newNRKNSession(ctx, opts.Headless)
	if err != nil {
		return logging.WithContext(fmt.Errorf("open session: %w", err), "session", "")
	}
	defer func() {
		cleanupStarted := log.Started("cleanup", "operation", "browser_close")
		if err := s.Close(); err != nil {
			log.Warning("cleanup", "browser close failed", "operation", "browser_close", "error", err)
			return
		}
		log.Completed("cleanup", cleanupStarted, "operation", "browser_close")
	}()
	log.Completed("session", sessionStarted, "headless", opts.Headless)
	fetchCtx, cancel := context.WithTimeout(ctx, 3*time.Minute)
	defer cancel()
	if opts.FIGIResolver == nil {
		opts.FIGIResolver = newNRKNFIGIResolver()
	}
	if opts.Logger == nil {
		opts.Logger = log.Raw()
	}
	fetchStarted := log.Started("fetch")
	a, fetchErr := nrkn.FetchAssets(fetchCtx, s, opts)
	if a == nil {
		if fetchErr == nil {
			fetchErr = logging.WithContext(fmt.Errorf("fetch assets returned no result"), "fetch", "")
		}
		return fetchErr
	}
	if fetchErr == nil {
		log.Completed("fetch", fetchStarted, "grand_total_jpy", a.GrandTotalJPY)
	}
	outputTarget := opts.OutputPath
	if outputTarget == "" {
		outputTarget = "stdout"
	}
	outputStarted := log.Started("output", "path", outputTarget)
	raw, err := cli.WriteNRKNJSON(r.stdout, opts.OutputPath, a)
	if err != nil {
		return logging.WithContext(fmt.Errorf("write assets json: %w", err), "output", "")
	}
	log.Completed("output", outputStarted, "path", outputTarget)
	if opts.S3Upload {
		uploadStarted := log.Started("s3_upload")
		key, err := cli.UploadNRKNJSON(ctx, opts.S3Client, opts.Now, raw)
		if err != nil {
			return logging.WithContext(fmt.Errorf("upload assets json: %w", err), "s3_upload", "")
		}
		log.Completed("s3_upload", uploadStarted, "key", key)
	}
	return fetchErr
}
