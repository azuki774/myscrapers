package cli

import (
	"errors"
	"flag"
	"io"
)

type Options struct {
	URL        string
	OutputPath string
	Headless   bool
}

func ParseArgs(args []string) (Options, error) {
	fs := flag.NewFlagSet("myscraper", flag.ContinueOnError)
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
