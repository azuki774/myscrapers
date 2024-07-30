package cmd

import (
	"context"
	"fmt"
	"log/slog"
	"myscrapers/internal/scenario"

	"github.com/spf13/cobra"
)

var downloadArgsOption downloadArgsOpt

type downloadArgsOpt struct {
	SiteName  string
	OutputDir string
	LastMonth bool
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
		sc, err := scenario.NewScenarioSBI()
		if err != nil {
			return err
		}
		if err = sc.Start(ctx); err != nil {
			slog.Error("failed to scrape", "err", err.Error())
			return err
		}
	case "moneyforward":
		mf, err := scenario.NewScenarioMoneyForward(downloadArgsOption.LastMonth)
		if err != nil {
			return err
		}
		if err = mf.Start(ctx); err != nil {
			slog.Error("failed to scrape", "err", err.Error())
			return err
		}
	case "test-github":
		sc := scenario.NewTestGitHub()
		return sc.Start(ctx)
	default:
		return fmt.Errorf("unknown website")
	}
	return nil
}

func init() {
	rootCmd.AddCommand(downloadCmd)
	downloadCmd.Flags().BoolVarP(&downloadArgsOption.LastMonth, "lastmonth", "l", true, "fetch last month") // 先月分も読み込むかどうか（月末取りこぼしを防ぐため、基本は true）
}
