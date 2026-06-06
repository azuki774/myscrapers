// Package cli parses command-line arguments for the myscraper binary and
// dispatches them to a scrape.Runner. It is kept independent of any
// concrete browser implementation so the wiring can be exercised from
// tests via the Runner interface.
package cli

import (
	"errors"
	"flag"
	"io"
)

// Options is the parsed view of the myscraper command-line flags.
// OutputPath has a default of "tmp/page.html" applied during flag
// parsing, so callers only need to fill in URL and Headless.
type Options struct {
	URL        string
	OutputPath string
	Headless   bool
}

// ParseArgs converts raw argv-style arguments into a validated Options.
// It returns an error if flag parsing fails or if the required --url
// flag is missing. OutputPath falls back to a repo-local default so
// the most common invocation does not need to specify it.
func ParseArgs(args []string) (Options, error) {
	fs := flag.NewFlagSet("myscraper", flag.ContinueOnError)
	// Discard flag's own usage output so parse errors are reported via
	// the returned error and can be written to a caller-controlled stream.
	fs.SetOutput(io.Discard)

	opts := Options{}
	fs.StringVar(&opts.URL, "url", "", "target URL to scrape")
	fs.StringVar(&opts.OutputPath, "out", "tmp/page.html", "path to write HTML snapshot")
	fs.BoolVar(&opts.Headless, "headless", true, "launch browser in headless mode")

	if err := fs.Parse(args); err != nil {
		return Options{}, err
	}
	if opts.URL == "" {
		return Options{}, errors.New("--url is required")
	}
	return opts, nil
}
