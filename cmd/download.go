package cmd

import (
	"context"
	"fmt"
	"log/slog"
	"os"

	"github.com/spf13/cobra"
)

type DownloadArgsOption struct {
	SiteName string
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
		return startDownload(downloadArgsOption.SiteName)
	},
}

func startDownload(siteName string) (err error) {
	_ = context.Background()

	switch siteName {
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

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// downloadCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// downloadCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}
