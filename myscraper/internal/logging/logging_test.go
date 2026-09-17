package logging

import (
	"bytes"
	"encoding/json"
	"errors"
	"log/slog"
	"testing"
)

func TestLoggerAddsCommonEventFieldsAndDuration(t *testing.T) {
	var output bytes.Buffer
	l := New(slog.New(slog.NewJSONHandler(&output, nil)), "sbi")
	started := l.Started("page", "page", "portfolio")
	l.Completed("page", started, "page", "portfolio")

	lines := bytes.Split(bytes.TrimSpace(output.Bytes()), []byte{'\n'})
	if len(lines) != 2 {
		t.Fatalf("records = %d, want 2: %s", len(lines), output.Bytes())
	}
	var record map[string]any
	if err := json.Unmarshal(lines[1], &record); err != nil {
		t.Fatalf("decode completed record: %v", err)
	}
	for key, want := range map[string]string{"scraper": "sbi", "stage": "page", "event": "completed"} {
		if got := record[key]; got != want {
			t.Errorf("%s = %v, want %q", key, got, want)
		}
	}
	if duration, ok := record["duration_ms"].(float64); !ok || duration < 0 {
		t.Errorf("duration_ms = %v, want a non-negative number", record["duration_ms"])
	}
}

func TestContextPreservesStageAndPageThroughWrapping(t *testing.T) {
	want := WithContext(errors.New("body failed"), "page", "portfolio")
	wrapped := errors.New("outer")
	wrapped = wrapForTest(wrapped, want)
	stage, page, ok := Context(wrapped)
	if !ok || stage != "page" || page != "portfolio" {
		t.Fatalf("context = %q/%q/%v, want page/portfolio/true", stage, page, ok)
	}
}

func wrapForTest(outer, inner error) error {
	return &testWrapper{outer: outer, inner: inner}
}

type testWrapper struct {
	outer error
	inner error
}

func (w *testWrapper) Error() string { return w.outer.Error() + ": " + w.inner.Error() }
func (w *testWrapper) Unwrap() error { return w.inner }
