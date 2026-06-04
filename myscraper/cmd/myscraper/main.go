// Command myscraper is the entry point for the v2 scraper CLI.
// It wires the standard flag-based CLI in package cli to a scrape.Service
// backed by the Playwright browser implementation, and exits with the
// status code returned by cli.Run.
package main

import (
	"os"

	"github.com/azuki774/myscrapers/myscraper/internal/browser"
	"github.com/azuki774/myscrapers/myscraper/internal/cli"
	"github.com/azuki774/myscrapers/myscraper/internal/scrape"
)

func main() {
	// Hand argv, stdout, and stderr straight to cli.Run so the CLI stays
	// testable in isolation. The concrete Browser dependency is injected
	// here so the cli package does not import the Playwright implementation.
	os.Exit(cli.Run(
		os.Args[1:],
		os.Stdout,
		os.Stderr,
		scrape.Service{Browser: browser.PlaywrightBrowser{}},
	))
}
