package main

import (
	"os"

	"github.com/azuki774/myscrapers/myscraper/internal/cli"
)

func main() {
	os.Exit(cli.Run(os.Stdout, os.Stderr))
}
