package cli

import (
	"bytes"
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
		s3Upload    bool
	)
	fs.StringVar(&passkeyPath, "passkey", envOr("SBI_PASSKEY_PATH", "/home/azuki/.local/state/opencode/sbi-passkey.json"), "path to the saved passkey JSON")
	fs.StringVar(&outPath, "output", envOr("SBI_OUTPUT", ""), "write JSON to this file instead of stdout")
	fs.BoolVar(&headless, "headless", true, "run browser headless")
	fs.BoolVar(&s3Upload, "s3-upload", false, "archive the emitted JSON to S3 after writing it")

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
		S3Upload:    s3Upload,
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
// regardless of how the assets value was built. It returns the exact
// bytes written (including the trailing newline) so callers can archive
// the same payload to S3 without re-marshaling.
func WriteAssetsJSON(w io.Writer, path string, assets *sbi.Assets) ([]byte, error) {
	assets.SchemaVersion = sbi.CurrentSchemaVersion
	raw, err := json.MarshalIndent(assets, "", "  ")
	if err != nil {
		return nil, err
	}
	payload := append(raw, '\n')
	if path == "" {
		if _, err := w.Write(payload); err != nil {
			return nil, err
		}
		return payload, nil
	}
	if err := os.WriteFile(path, payload, 0o600); err != nil {
		return nil, err
	}
	return payload, nil
}

// UploadAssetsJSON archives the already-marshaled assets JSON to S3,
// returning the object key it was written to. It reuses the same bytes
// emitted by WriteAssetsJSON so stdout/file and S3 stay in lockstep.
func UploadAssetsJSON(ctx context.Context, client sbi.S3Client, now time.Time, raw []byte) (string, error) {
	key := client.KeyForTime(now)
	if err := client.PutJSON(ctx, key, bytes.NewReader(raw)); err != nil {
		return "", err
	}
	return key, nil
}
