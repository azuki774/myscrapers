package cmd

import (
	"context"
	"fmt"
	"log/slog"
	"os"

	"github.com/spf13/cobra"
)

type DownloadArgsOption struct {
	SiteName  string
	OutputDir string
}

var downloadArgsOption DownloadArgsOption

// downloadCmd represents the download command
var downloadCmd = &cobra.Command{
	Use:   "download",
	Short: "Download from website",
	Long:  `Download from website`,
	RunE: func(cmd *cobra.Command, args []string) error {
		slog.Info("download called")
		if len(args) == 0 {
			return fmt.Errorf("required website args")
		}
		downloadArgsOption.SiteName = args[0]

		return startDownload(downloadArgsOption)
	},
}

func startDownload(opts DownloadArgsOption) (err error) {
	_ = context.Background()
	slog.Info("show config", "sitename", opts.SiteName, "outputDir", opts.OutputDir)
	switch opts.SiteName {
	case "sbi":

	default:
		return fmt.Errorf("unknown website")
	}

	return nil
}

func init() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	slog.SetDefault(logger)
	rootCmd.AddCommand(downloadCmd)

	// Get current path
	currPath, err := os.Getwd()
	if err != nil {
		panic(err)
	}

	downloadCmd.Flags().StringVarP(&downloadArgsOption.OutputDir, "output", "o", currPath, "output directory")
}
