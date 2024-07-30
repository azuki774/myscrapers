package cmd

import (
	"context"
	"fmt"
	"myscrapers/internal/importer"

	"log/slog"

	"github.com/spf13/cobra"
)

var ImportArgsOption importArgsOpt

type importArgsOpt struct {
	OpeName string
}

// importCmd represents the import command
var importCmd = &cobra.Command{
	Use:   "import",
	Short: "A brief description of your command",
	Long: `A longer description that spans multiple lines and likely contains examples
and usage of using your command. For example:

Cobra is a CLI library for Go that empowers applications.
This application is a tool to generate the needed files
to quickly create a Cobra application.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		slog.Info("import called")
		if len(args) == 0 {
			return fmt.Errorf("required website args")
		}
		ImportArgsOption.OpeName = args[0]

		return startImport(ImportArgsOption)
	},
}

func startImport(opts importArgsOpt) (err error) {
	ctx := context.Background()
	slog.Info("show config", "opeName", opts.OpeName)
	switch opts.OpeName {
	case "moneyforward-cf":
		cf, err := importer.NewImporterCF(ctx)
		if err != nil {
			return err
		}
		if err = cf.Start(ctx); err != nil {
			slog.Error("failed to scrape", "err", err.Error())
			return err
		}
	default:
		return fmt.Errorf("unknown website")
	}
	return nil
}

func init() {
	rootCmd.AddCommand(importCmd)
}
