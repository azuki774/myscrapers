package cli

import (
	"flag"
	"fmt"
	"io"
)

func newFlagSet(stderr io.Writer) *flag.FlagSet {
	fs := flag.NewFlagSet("myscraper", flag.ContinueOnError)
	fs.SetOutput(stderr)
	fs.Usage = func() {
		fmt.Fprintf(stderr, "usage: myscraper moneyforward [--fetch|--update] [flags]\n")
		fs.PrintDefaults()
	}
	return fs
}
