package cli

import (
	"context"
	"io"
	"log/slog"

	"github.com/azuki774/myscrapers/myscraper/internal/scrape"
)

type Runner interface {
	Run(ctx context.Context, req scrape.Request) (scrape.Result, error)
}

func Run(args []string, stdout, stderr io.Writer, logger *slog.Logger, runner Runner) int {
	if logger == nil {
		logger = slog.New(slog.NewJSONHandler(io.Discard, nil))
	}
	opts, err := ParseArgs(args)
	if err != nil {
		logger.Error("argument parsing failed", "error", err)
		return 2
	}

	logger.Info("starting scrape", "url", opts.URL, "output", opts.OutputPath)
	result, err := runner.Run(context.Background(), scrape.Request{
		URL:        opts.URL,
		OutputPath: opts.OutputPath,
		Headless:   opts.Headless,
	})
	if err != nil {
		logger.Error("scrape failed", "error", err)
		return 1
	}

	logger.Info("scrape complete", "output", result.OutputPath, "title", result.Title)
	return 0
}
