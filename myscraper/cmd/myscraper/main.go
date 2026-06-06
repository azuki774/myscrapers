// Command myscraper is the entry point for the v2 scraper CLI.
// It dispatches the optional `moneyforward` subcommand to the moneyforward
// CLI; all other invocations fall through to the URL-scrape flow backed
// by a Playwright browser. Exits with the status code returned by the
// subcommand.
package main

import (
	"log/slog"
	"os"

	"github.com/azuki774/myscrapers/myscraper/internal/browser"
	"github.com/azuki774/myscrapers/myscraper/internal/cli"
	"github.com/azuki774/myscrapers/myscraper/internal/scrape"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	if len(os.Args) > 1 && os.Args[1] == "moneyforward" {
		os.Exit(cli.RunMoneyforward(
			os.Args[1:],
			os.Stdout,
			os.Stderr,
			logger,
			moneyforwardRunner{logger: logger, headless: true},
		))
	}
	os.Exit(cli.Run(
		os.Args[1:],
		os.Stdout,
		os.Stderr,
		scrape.Service{Browser: browser.PlaywrightBrowser{}},
	))
}
