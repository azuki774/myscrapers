package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"testing"

	"github.com/azuki774/myscrapers/myscraper/internal/logging"
	"github.com/azuki774/myscrapers/myscraper/internal/nrkn"
)

func TestRunSBITerminalErrorContainsFailurePageOnce(t *testing.T) {
	passkey := writePasskeyFile(t, t.TempDir())
	var logs bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&logs, nil))
	runner := &fakeSBIRunner{err: logging.WithContext(errors.New("body failed"), "page", "portfolio")}
	if code := RunSBI([]string{"sbi", "--passkey", passkey}, &bytes.Buffer{}, &bytes.Buffer{}, logger, runner); code != 1 {
		t.Fatalf("exit = %d, want 1", code)
	}
	var errorsFound []map[string]any
	decoder := json.NewDecoder(&logs)
	for {
		var record map[string]any
		err := decoder.Decode(&record)
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatalf("decode log: %v", err)
		}
		if record["level"] == "ERROR" {
			errorsFound = append(errorsFound, record)
		}
	}
	if len(errorsFound) != 1 {
		t.Fatalf("terminal ERROR records = %d, want 1: %s", len(errorsFound), logs.String())
	}
	if errorsFound[0]["stage"] != "page" || errorsFound[0]["page"] != "portfolio" {
		t.Fatalf("terminal context = stage=%v page=%v, want page/portfolio", errorsFound[0]["stage"], errorsFound[0]["page"])
	}
}

func TestRunNRKNFailureUsesSingleTerminalError(t *testing.T) {
	t.Setenv("NRKN_ID", "id")
	t.Setenv("NRKN_PASS", "pass")
	t.Setenv("NRKN_BIRTHDAY", "20000101")
	var logs bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&logs, nil))
	runner := nrknErrorRunner{}
	if code := RunNRKN([]string{"nrkn"}, &bytes.Buffer{}, &bytes.Buffer{}, logger, runner); code != 1 {
		t.Fatalf("exit = %d, want 1", code)
	}
	var terminalErrors int
	decoder := json.NewDecoder(&logs)
	for {
		var record map[string]any
		err := decoder.Decode(&record)
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatalf("decode log: %v", err)
		}
		if record["level"] == "ERROR" {
			terminalErrors++
			if record["stage"] != "login" {
				t.Errorf("stage = %v, want login", record["stage"])
			}
		}
	}
	if terminalErrors != 1 {
		t.Fatalf("terminal ERROR records = %d, want 1: %s", terminalErrors, logs.String())
	}
}

type nrknErrorRunner struct{}

func (nrknErrorRunner) RunAssets(context.Context, nrkn.FetchOptions) error {
	return logging.WithContext(errors.New("login failed"), "login", "")
}
