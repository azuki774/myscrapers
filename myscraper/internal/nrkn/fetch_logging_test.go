package nrkn

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"testing"
)

func TestFetchAssetsLogsCommonStagesAndRetry(t *testing.T) {
	var output bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&output, nil))
	f := &fakeSession{body: resolverFixture(), loginErrs: []error{ErrConcurrentLogin}}
	o := testOptions()
	o.Logger = logger
	if _, err := FetchAssets(context.Background(), f, o); err != nil {
		t.Fatalf("FetchAssets: %v", err)
	}

	var records []map[string]any
	decoder := json.NewDecoder(&output)
	for {
		var record map[string]any
		err := decoder.Decode(&record)
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatalf("decode log: %v", err)
		}
		records = append(records, record)
	}
	for _, stage := range []string{"login", "page", "resolve_figi", "cleanup"} {
		if !hasCompletedStage(records, stage) {
			t.Errorf("stage %q has no completed record", stage)
		}
	}
	if !hasRecord(records, "login", "warning") {
		t.Error("concurrent login retry has no warning record")
	}
	for _, record := range records {
		if record["scraper"] != "nrkn" {
			t.Errorf("scraper = %v, want nrkn", record["scraper"])
		}
	}
}

func TestFetchAssetsFailedPageHasNoCompletedRecord(t *testing.T) {
	var output bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&output, nil))
	f := &fakeSession{body: "<html><body>not an asset page</body></html>"}
	o := testOptions()
	o.Logger = logger
	if _, err := FetchAssets(context.Background(), f, o); err == nil {
		t.Fatal("malformed asset page should fail")
	}

	var records []map[string]any
	decoder := json.NewDecoder(&output)
	for {
		var record map[string]any
		err := decoder.Decode(&record)
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatalf("decode log: %v", err)
		}
		records = append(records, record)
	}
	if !hasRecord(records, "page", "started") {
		t.Fatal("page started record missing")
	}
	if hasRecord(records, "page", "completed") {
		t.Fatal("page completed record emitted after page failure")
	}
	if !hasCompletedStage(records, "cleanup") {
		t.Fatal("cleanup completion record missing")
	}
}

func TestFetchAssetsLogoutFailureHasNoCleanupCompletion(t *testing.T) {
	var output bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&output, nil))
	f := &fakeSession{body: resolverFixture(), logoutErr: errors.New("logout unavailable")}
	o := testOptions()
	o.Logger = logger
	assets, err := FetchAssets(context.Background(), f, o)
	if assets == nil || err == nil {
		t.Fatalf("assets=%v err=%v, want assets plus cleanup error", assets, err)
	}

	var records []map[string]any
	decoder := json.NewDecoder(&output)
	for {
		var record map[string]any
		err := decoder.Decode(&record)
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatalf("decode log: %v", err)
		}
		records = append(records, record)
	}
	if !hasRecord(records, "cleanup", "warning") {
		t.Fatal("cleanup warning record missing")
	}
	if hasRecord(records, "cleanup", "completed") {
		t.Fatal("cleanup completed record emitted after logout failure")
	}
}

func hasCompletedStage(records []map[string]any, stage string) bool {
	return hasRecord(records, stage, "completed")
}

func hasRecord(records []map[string]any, stage, event string) bool {
	for _, record := range records {
		if record["stage"] == stage && record["event"] == event {
			return true
		}
	}
	return false
}
