package cli

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"os"
	"time"

	"github.com/azuki774/myscrapers/myscraper/internal/sbi"
)

// SBIRunner is the contract the sbi CLI depends on. Production wires
// it to a runner that opens a Playwright session, logs in with the
// saved passkey, and fetches the asset summary.
type SBIRunner interface {
	RunAssets(ctx context.Context, opts sbi.FetchOptions) error
}

// RunSBI parses the "sbi" subcommand argv slice, builds
// sbi.FetchOptions, dispatches to the injected runner, and returns the
// conventional exit code (0 success, 1 runner error, 2 argument error).
func RunSBI(
	args []string,
	stdout, stderr io.Writer,
	logger *slog.Logger,
	runner SBIRunner,
) int {
	fs := newFlagSet(stderr)
	var (
		passkeyPath string
		outPath     string
		headless    bool
	)
	fs.StringVar(&passkeyPath, "passkey", envOr("SBI_PASSKEY_PATH", "/home/azuki/.local/state/opencode/sbi-passkey.json"), "path to the saved passkey JSON")
	fs.StringVar(&outPath, "output", envOr("SBI_OUTPUT", ""), "write JSON to this file instead of stdout")
	fs.BoolVar(&headless, "headless", true, "run browser headless")

	if err := fs.Parse(args[1:]); err != nil {
		return 2
	}

	passkey, err := sbi.LoadPasskey(passkeyPath)
	if err != nil {
		logger.Error("failed to load passkey", "error", err, "path", passkeyPath)
		return 1
	}

	opts := sbi.FetchOptions{
		PasskeyPath: passkeyPath,
		Passkey:     passkey,
		OutputPath:  outPath,
		Now:         time.Now(),
		Logger:      logger,
		Headless:    headless,
	}
	if err := runner.RunAssets(context.Background(), opts); err != nil {
		logger.Error("sbi fetch failed", "error", err)
		return 1
	}
	return 0
}

// WriteAssetsJSON marshals the asset summary and writes it either to
// the given file or to stdout. It stamps the current schema version so
// every emitted JSON carries schema_version as its first key,
// regardless of how the assets value was built.
func WriteAssetsJSON(w io.Writer, path string, assets *sbi.Assets) error {
	assets.SchemaVersion = sbi.CurrentSchemaVersion
	raw, err := json.MarshalIndent(assets, "", "  ")
	if err != nil {
		return err
	}
	if path == "" {
		_, err = w.Write(append(raw, '\n'))
		return err
	}
	return os.WriteFile(path, append(raw, '\n'), 0o600)
}
