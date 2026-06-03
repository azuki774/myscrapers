package cli

import (
	"context"
	"fmt"
	"io"

	"github.com/azuki774/myscrapers/myscraper/internal/scrape"
)

type Runner interface {
	Run(ctx context.Context, req scrape.Request) (scrape.Result, error)
}

func Run(args []string, stdout, stderr io.Writer, runner Runner) int {
	opts, err := ParseArgs(args)
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 2
	}

	result, err := runner.Run(context.Background(), scrape.Request{
		URL:        opts.URL,
		OutputPath: opts.OutputPath,
		Headless:   opts.Headless,
	})
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}

	fmt.Fprintf(stdout, "saved %s (%s)\n", result.OutputPath, result.Title)
	return 0
}
