package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"time"

	"github.com/azuki774/myscrapers/myscraper/internal/logging"
	"github.com/azuki774/myscrapers/myscraper/internal/nrkn"
	"github.com/azuki774/myscrapers/myscraper/internal/storage"
)

type NRKNRunner interface {
	RunAssets(context.Context, nrkn.FetchOptions) error
}
type nrknStore interface{ nrkn.S3Client }

var buildNRKNS3Store = func(ctx context.Context) (nrknStore, error) { return storage.New(ctx) }

func RunNRKN(args []string, stdout, stderr io.Writer, logger *slog.Logger, runner NRKNRunner) int {
	log := logging.New(logger, "nrkn")
	fs := newFlagSet(stderr)
	output := envOr("NRKN_OUTPUT", "")
	headless, upload := true, false
	fs.StringVar(&output, "output", output, "write JSON to this file instead of stdout")
	fs.BoolVar(&headless, "headless", true, "run browser headless")
	fs.BoolVar(&upload, "s3-upload", false, "archive emitted JSON to S3")
	if err := fs.Parse(args[1:]); err != nil {
		return 2
	}
	outputTarget := output
	if outputTarget == "" {
		outputTarget = "stdout"
	}
	runStarted := log.Started("run", "output", outputTarget, "s3_upload", upload)
	failRun := func(err error) int {
		stage, page, ok := logging.Context(err)
		if !ok || stage == "" {
			stage = "run"
		}
		log.Error(stage, page, err)
		return 1
	}
	credentialsStarted := log.Started("credentials", "source", "environment")
	creds := nrkn.Credentials{ID: os.Getenv("NRKN_ID"), Password: os.Getenv("NRKN_PASS"), Birthday: os.Getenv("NRKN_BIRTHDAY")}
	if creds.ID == "" || creds.Password == "" || creds.Birthday == "" {
		log.Error("credentials", "", fmt.Errorf("NRKN_ID, NRKN_PASS, and NRKN_BIRTHDAY are required"))
		return 2
	}
	log.Completed("credentials", credentialsStarted, "source", "environment")
	var client nrkn.S3Client
	if upload {
		storageStarted := log.Started("storage", "operation", "build_store")
		store, err := buildNRKNS3Store(context.Background())
		if err != nil {
			return failRun(logging.WithContext(fmt.Errorf("build S3 store: %w", err), "storage", ""))
		}
		log.Completed("storage", storageStarted, "operation", "build_store")
		client = store
	}
	err := runner.RunAssets(context.Background(), nrkn.FetchOptions{Credentials: creds, OutputPath: output, Now: time.Now(), Logger: logger, Headless: headless, S3Upload: upload, S3Client: client})
	if err != nil {
		return failRun(err)
	}
	log.Completed("run", runStarted, "output", outputTarget, "s3_upload", upload)
	return 0
}

func WriteNRKNJSON(w io.Writer, path string, a *nrkn.Assets) ([]byte, error) {
	a.SchemaVersion = nrkn.CurrentSchemaVersion
	raw, err := json.MarshalIndent(a, "", "  ")
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
	dir := filepath.Dir(path)
	tmp, err := os.CreateTemp(dir, ".nrkn-*.json")
	if err != nil {
		return nil, err
	}
	tmpName := tmp.Name()
	defer os.Remove(tmpName)
	if err := tmp.Chmod(0o600); err != nil {
		_ = tmp.Close()
		return nil, err
	}
	if _, err := tmp.Write(payload); err != nil {
		_ = tmp.Close()
		return nil, err
	}
	if err := tmp.Close(); err != nil {
		return nil, err
	}
	if err := os.Rename(tmpName, path); err != nil {
		return nil, err
	}
	return payload, nil
}

func UploadNRKNJSON(ctx context.Context, client nrkn.S3Client, now time.Time, raw []byte) (string, error) {
	if client == nil {
		return "", fmt.Errorf("S3 client is required")
	}
	key := client.KeyForTime(now)
	if err := client.PutJSON(ctx, key, bytes.NewReader(raw)); err != nil {
		return "", err
	}
	return key, nil
}
