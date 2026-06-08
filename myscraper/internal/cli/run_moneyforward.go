package cli

import (
	"context"
	"io"
	"log/slog"
	"os"
	"time"

	"github.com/azuki774/myscrapers/myscraper/internal/moneyforward"
)

// MoneyforwardRunner is the contract the moneyforward CLI depends on.
// Production wires it to a concrete runner that opens a Playwright
// session and dispatches to moneyforward.Fetch / moneyforward.Update.
// Tests use a fake.
type MoneyforwardRunner interface {
	RunFetch(ctx context.Context, opts moneyforward.FetchOptions, s3Upload bool) error
	RunUpdate(ctx context.Context, opts moneyforward.UpdateOptions) error
	RunFetchCookie(ctx context.Context, cookiePath string) error
}

// RunMoneyforward parses the "moneyforward" subcommand argv slice,
// builds a FetchOptions / UpdateOptions struct from the environment,
// dispatches to the injected runner, and returns the conventional exit
// code (0 success, 1 runner error, 2 argument error).
//
// argv is expected to look like: ["moneyforward", "--fetch|--update",
// ...flags]. The first element ("moneyforward") is consumed for
// symmetry with the dispatcher in cmd/myscraper/main.go.
func RunMoneyforward(
	args []string,
	stdout, stderr io.Writer,
	logger *slog.Logger,
	runner MoneyforwardRunner,
) int {
	if len(args) > 1 && args[1] == "fetch-cookie" {
		fs := newFlagSet(stderr)
		cookieP := fs.String("cookie-path", envOr("MF_COOKIE_PATH", "/data/cookie.json"), "cookie JSON destination path")
		if err := fs.Parse(args[2:]); err != nil {
			return 2
		}
		if err := runner.RunFetchCookie(context.Background(), *cookieP); err != nil {
			logger.Error("fetch-cookie failed", "error", err)
			return 1
		}
		logger.Info("fetch-cookie complete", "path", *cookieP)
		return 0
	}

	fs := newFlagSet(stderr)
	var (
		fetch    bool
		update   bool
		s3Upload bool
		cookieP  string
		outDir   string
		headless bool
	)
	fs.BoolVar(&fetch, "fetch", false, "scrape CF and asset history and write CSVs")
	fs.BoolVar(&update, "update", false, "press 一括更新 and モバイルSuica 更新")
	fs.BoolVar(&s3Upload, "s3-upload", false, "(with --fetch) upload the three CSVs after writing")
	fs.StringVar(&cookieP, "cookie-path", envOr("MF_COOKIE_PATH", "/data/cookie.json"), "cookie JSON path")
	fs.StringVar(&outDir, "output-dir", envOr("MF_OUTPUT_DIR", "/data"), "output directory")
	fs.BoolVar(&headless, "headless", true, "run browser headless")

	if err := fs.Parse(args[1:]); err != nil {
		return 2
	}
	if (fetch && update) || (!fetch && !update) {
		logger.Error("exactly one of --fetch or --update is required")
		return 2
	}
	cookies, err := moneyforward.LoadCookies(cookieP)
	if err != nil {
		logger.Error("failed to load cookies", "error", err, "path", cookieP)
		return 1
	}
	if fetch {
		opts := moneyforward.FetchOptions{
			Session:   nil,
			Cookies:   cookies,
			OutputDir: outDir,
			Now:       time.Now(),
			Logger:    logger,
		}
		if err := runner.RunFetch(context.Background(), opts, s3Upload); err != nil {
			logger.Error("fetch failed", "error", err)
			return 1
		}
		logger.Info("fetch complete")
		return 0
	}
	opts := moneyforward.UpdateOptions{
		Session: nil,
		Cookies: cookies,
		Logger:  logger,
	}
	if s3Upload {
		logger.Warn("--s3-upload has no effect with --update; ignoring")
	}
	_ = headless
	if err := runner.RunUpdate(context.Background(), opts); err != nil {
		logger.Error("update failed", "error", err)
		return 1
	}
	logger.Info("update complete")
	return 0
}

func envOr(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}
