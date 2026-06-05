// Package cli parses command-line arguments for the myscraper binary and
// dispatches them to a scrape.Runner. It is kept independent of any
// concrete browser implementation so the wiring can be exercised from
// tests via the Runner interface.
package cli

import (
	"errors"
	"flag"
	"io"

	"github.com/azuki774/myscrapers/myscraper/internal/scrape"
	"github.com/azuki774/myscrapers/myscraper/internal/smbccard"
)

// ModeFetchURL is the default mode: fetch a single URL anonymously and
// dump its HTML snapshot. It preserves the v1 CLI shape. The
// authoritative value lives in package scrape to avoid a cli↔scrape
// import cycle; this constant is a re-export so existing callers
// (and tests in this package) keep their cli-prefix-free spelling.
const ModeFetchURL = scrape.ModeFetchURL

// Options is the parsed view of the myscraper command-line flags.
// OutputPath falls back to a per-mode default during flag parsing, so
// callers only need to fill in URL (for fetch-url) or nothing (for
// authenticated modes that decide their own output path).
type Options struct {
	Mode       string
	URL        string
	OutputPath string
	Headless   bool
}

// ParseArgs converts raw argv-style arguments into a validated Options.
// It returns an error if flag parsing fails, if the mode is unknown,
// or if the required per-mode inputs are missing. OutputPath is filled
// with a mode-specific default when the caller does not supply one.
func ParseArgs(args []string) (Options, error) {
	fs := flag.NewFlagSet("myscraper", flag.ContinueOnError)
	// Discard flag's own usage output so parse errors are reported via
	// the returned error and can be written to a caller-controlled stream.
	fs.SetOutput(io.Discard)

	opts := Options{}
	fs.StringVar(&opts.Mode, "mode", ModeFetchURL, "run mode")
	fs.StringVar(&opts.URL, "url", "", "target URL to scrape when --mode=fetch-url")
	fs.StringVar(&opts.OutputPath, "out", "", "path to write HTML snapshot")
	fs.BoolVar(&opts.Headless, "headless", true, "launch browser in headless mode")

	if err := fs.Parse(args); err != nil {
		return Options{}, err
	}

	switch opts.Mode {
	case ModeFetchURL:
		if opts.URL == "" {
			return Options{}, errors.New("--url is required when --mode=fetch-url")
		}
		if opts.OutputPath == "" {
			opts.OutputPath = "tmp/page.html"
		}
	case smbccard.ModeWebMeisaiHTMLDump:
		if opts.OutputPath == "" {
			opts.OutputPath = "tmp/smbccard-webmeisai.html"
		}
	default:
		return Options{}, errors.New("unsupported --mode: " + opts.Mode)
	}

	return opts, nil
}
