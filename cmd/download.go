package cmd

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"scraper-go/internal/scenario"

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
	slog.Info("show config", "sitename", opts.SiteName, "outputDir", opts.OutputDir)
	switch opts.SiteName {
	case "sbi":
		user := os.Getenv("user")
		pass := os.Getenv("pass")
		sc, err := scenario.NewScenarioSBI(downloadArgsOption.OutputDir, user, pass)
		if err != nil {
			slog.Error("failed to create scenario sbi", slog.String("error", err.Error()))
			return err
		}
		slog.Info("success scenario sbi")
		return sc.Start(ctx)
	default:
		return fmt.Errorf("unknown website")
	}

}

func init() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{AddSource: true}))
	slog.SetDefault(logger)
	rootCmd.AddCommand(downloadCmd)

	// Get current path
	currPath, err := os.Getwd()
	if err != nil {
		panic(err)
	}

	downloadCmd.Flags().StringVarP(&downloadArgsOption.OutputDir, "output", "o", currPath, "output directory")
}
