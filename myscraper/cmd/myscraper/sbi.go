package main

import (
	"context"
	"fmt"
	"io"
	"log/slog"

	"github.com/azuki774/myscrapers/myscraper/internal/cli"
	"github.com/azuki774/myscrapers/myscraper/internal/logging"
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
	log := logging.New(r.logger, "sbi")
	sessionStarted := log.Started("session", "headless", opts.Headless)
	sess, err := sbi.NewPlaywrightSession(ctx, opts.Headless)
	if err != nil {
		return logging.WithContext(fmt.Errorf("open session: %w", err), "session", "")
	}
	defer func() {
		cleanupStarted := log.Started("cleanup", "operation", "browser_close")
		if err := sess.Close(); err != nil {
			log.Warning("cleanup", "browser close failed", "operation", "browser_close", "error", err)
			return
		}
		log.Completed("cleanup", cleanupStarted, "operation", "browser_close")
	}()
	log.Completed("session", sessionStarted, "headless", opts.Headless)

	loginStarted := log.Started("login")
	if err := sess.LoginWithPasskey(ctx, opts.Passkey); err != nil {
		return logging.WithContext(fmt.Errorf("passkey login: %w", err), "login", "")
	}
	log.Completed("login", loginStarted)

	fetchStarted := log.Started("fetch")
	fetchLogger := opts.Logger
	if fetchLogger == nil {
		fetchLogger = log.Raw()
	}
	assets, err := sbi.FetchAssets(ctx, sess, opts.Now, sbi.NewOpenFIGIResolver(nil), fetchLogger)
	if err != nil {
		if _, _, ok := logging.Context(err); !ok {
			err = logging.WithContext(fmt.Errorf("fetch assets: %w", err), "fetch", "")
		}
		return err
	}
	log.Completed("fetch", fetchStarted, "grand_total_jpy", assets.GrandTotalJPY)

	outputTarget := opts.OutputPath
	if outputTarget == "" {
		outputTarget = "stdout"
	}
	outputStarted := log.Started("output", "path", outputTarget)
	raw, err := cli.WriteAssetsJSON(r.stdout, opts.OutputPath, assets)
	if err != nil {
		return logging.WithContext(fmt.Errorf("write assets json: %w", err), "output", "")
	}
	log.Completed("output", outputStarted, "path", outputTarget)
	if opts.S3Upload {
		if opts.S3Client == nil {
			return logging.WithContext(fmt.Errorf("s3 upload requested but no S3 client configured"), "s3_upload", "")
		}
		uploadStarted := log.Started("s3_upload")
		key, err := cli.UploadAssetsJSON(ctx, opts.S3Client, opts.Now, raw)
		if err != nil {
			return logging.WithContext(fmt.Errorf("upload assets json: %w", err), "s3_upload", "")
		}
		log.Completed("s3_upload", uploadStarted, "key", key)
	}
	return nil
}
