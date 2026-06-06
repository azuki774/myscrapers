package cli

import (
	"context"
	"fmt"
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
		fmt.Fprintln(stderr, "exactly one of --fetch or --update is required")
		return 2
	}
	cookies, err := moneyforward.LoadCookies(cookieP)
	if err != nil {
		fmt.Fprintf(stderr, "load cookies: %v\n", err)
		return 1
	}
	if fetch {
		opts := moneyforward.FetchOptions{
			Session:   nil, // runner opens the session
			Cookies:   cookies,
			OutputDir: outDir,
			Now:       time.Now(),
			Logger:    logger,
		}
		if err := runner.RunFetch(context.Background(), opts, s3Upload); err != nil {
			fmt.Fprintf(stderr, "fetch: %v\n", err)
			return 1
		}
		fmt.Fprintln(stdout, "fetch complete")
		return 0
	}
	opts := moneyforward.UpdateOptions{
		Session: nil, // runner opens the session
		Cookies: cookies,
		Logger:  logger,
	}
	if s3Upload {
		logger.Warn("--s3-upload has no effect with --update; ignoring")
	}
	_ = headless // currently always true; flag is plumbed for future use
	if err := runner.RunUpdate(context.Background(), opts); err != nil {
		fmt.Fprintf(stderr, "update: %v\n", err)
		return 1
	}
	fmt.Fprintln(stdout, "update complete")
	return 0
}

func envOr(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}
