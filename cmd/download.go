package cmd

import (
	"context"
	"fmt"
	"log/slog"
	"myscrapers/internal/scenario"
	"os"

	"github.com/spf13/cobra"
)

var downloadArgsOption downloadArgsOpt

type downloadArgsOpt struct {
	SiteName  string
	OutputDir string
}

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

func startDownload(opts downloadArgsOpt) (err error) {
	ctx := context.Background()
	slog.Info("show config", "sitename", opts.SiteName)
	switch opts.SiteName {
	case "sbi":
		fmt.Println("not yet implemented")
		return fmt.Errorf("not yet implemented")
	case "test-github":
		sc := scenario.NewTestGitHub()
		return sc.Start(ctx)
	default:
		return fmt.Errorf("unknown website")
	}
}

func init() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{AddSource: true}))
	slog.SetDefault(logger)
	rootCmd.AddCommand(downloadCmd)
}
