package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log/slog"
	"os"
	"time"

	"github.com/azuki774/myscrapers/myscraper/internal/sbi"
	"github.com/azuki774/myscrapers/myscraper/internal/storage"
)

// SBIRunner is the contract the sbi CLI depends on. Production wires
// it to a runner that opens a Playwright session, logs in with the
// saved passkey, and fetches the asset summary.
type SBIRunner interface {
	RunAssets(ctx context.Context, opts sbi.FetchOptions) error
}

// s3Store is the S3 surface the sbi CLI needs in S3 mode: it must both
// archive the result JSON (sbi.S3Client) and fetch the passkey bundle.
// storage.Store satisfies it.
type s3Store interface {
	sbi.S3Client
	Download(ctx context.Context, key, destPath string) error
}

// buildS3Store constructs the S3 store from the environment. It is a
// variable so tests can inject a fake.
var buildS3Store = func(ctx context.Context) (s3Store, error) {
	return storage.New(ctx)
}

// defaultPasskeyPath is used when neither --passkey nor SBI_PASSKEY_PATH
// is supplied (local mode).
const defaultPasskeyPath = "/home/azuki/.local/state/opencode/sbi-passkey.json"

// RunSBI parses the "sbi" subcommand argv slice, builds
// sbi.FetchOptions, dispatches to the injected runner, and returns the
// conventional exit code (0 success, 1 runner error, 2 argument error).
//
// In local mode (no --s3-upload) the passkey is read from --passkey /
// SBI_PASSKEY_PATH / the default local path, and the result is written
// to --output or stdout. In S3 mode (--s3-upload) the passkey is
// downloaded from BUCKET_DIR/passkey.json into a temp file and the
// result is additionally archived to S3; --passkey is ignored.
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
	fs.StringVar(&passkeyPath, "passkey", envOr("SBI_PASSKEY_PATH", defaultPasskeyPath), "path to the saved passkey JSON (local mode)")
	fs.StringVar(&outPath, "output", envOr("SBI_OUTPUT", ""), "write JSON to this file instead of stdout")
	fs.BoolVar(&headless, "headless", true, "run browser headless")
	fs.BoolVar(&s3Upload, "s3-upload", false, "fetch the passkey from S3 and archive the emitted JSON to S3")

	if err := fs.Parse(args[1:]); err != nil {
		return 2
	}

	var (
		resolvedPasskeyPath string
		passkey             *sbi.PasskeyFile
		s3Client            sbi.S3Client
	)
	if s3Upload {
		store, err := buildS3Store(context.Background())
		if err != nil {
			logger.Error("failed to build S3 store", "error", err)
			return 1
		}
		s3Client = store
		pkPath, pf, err := downloadPasskey(context.Background(), store, logger)
		if err != nil {
			logger.Error("failed to obtain passkey", "error", err)
			return 1
		}
		// The passkey was fetched from S3 into a temp file for this
		// run only. Delete it now so the secret does not linger on disk
		// after the process exits (success or failure).
		defer os.Remove(pkPath)
		resolvedPasskeyPath = pkPath
		passkey = pf
		if passkeyExplicitlySet(fs) {
			logger.Warn("--passkey is ignored in S3 mode; using S3 passkey", "path", passkeyPath)
		}
	} else {
		pk, err := sbi.LoadPasskey(passkeyPath)
		if err != nil {
			logger.Error("failed to load passkey", "error", err, "path", passkeyPath)
			return 1
		}
		resolvedPasskeyPath = passkeyPath
		passkey = pk
	}

	opts := sbi.FetchOptions{
		PasskeyPath: resolvedPasskeyPath,
		Passkey:     passkey,
		OutputPath:  outPath,
		Now:         time.Now(),
		Logger:      logger,
		Headless:    headless,
		S3Upload:    s3Upload,
		S3Client:    s3Client,
	}
	if err := runner.RunAssets(context.Background(), opts); err != nil {
		logger.Error("sbi fetch failed", "error", err)
		return 1
	}
	return 0
}

// passkeyExplicitlySet reports whether --passkey was passed on the
// command line (as opposed to being filled from SBI_PASSKEY_PATH or the
// default).
func passkeyExplicitlySet(fs *flag.FlagSet) bool {
	set := false
	fs.Visit(func(f *flag.Flag) {
		if f.Name == "passkey" {
			set = true
		}
	})
	return set
}

// downloadPasskey pulls BUCKET_DIR/passkey.json into a 0600 temp file,
// validates it, and returns the temp path plus the parsed passkey. The
// caller is responsible for removing the temp file.
func downloadPasskey(ctx context.Context, store s3Store, logger *slog.Logger) (string, *sbi.PasskeyFile, error) {
	tmp, err := os.CreateTemp("", "sbi-passkey-*.json")
	if err != nil {
		return "", nil, fmt.Errorf("create temp passkey file: %w", err)
	}
	tmpPath := tmp.Name()
	tmp.Close()
	logger.Info("downloading passkey from S3", "key", sbi.PasskeyS3Key, "dest", tmpPath)
	if err := store.Download(ctx, sbi.PasskeyS3Key, tmpPath); err != nil {
		os.Remove(tmpPath)
		return "", nil, fmt.Errorf("download passkey: %w", err)
	}
	pf, err := sbi.LoadPasskey(tmpPath)
	if err != nil {
		os.Remove(tmpPath)
		return "", nil, fmt.Errorf("load downloaded passkey: %w", err)
	}
	return tmpPath, pf, nil
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
