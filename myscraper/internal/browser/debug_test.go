package browser

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestFormatResponseLine(t *testing.T) {
	line := formatResponseLine(
		"2026-06-05T00:00:00Z",
		"https://www.smbc-card.com/index.jsp",
		403,
		"text/html",
		1234,
		"<html><body>Access Denied</body></html>",
	)
	var obj map[string]any
	if err := json.Unmarshal([]byte(line), &obj); err != nil {
		t.Fatalf("Unmarshal: %v; line=%q", err, line)
	}
	if got := obj["event"]; got != "response" {
		t.Fatalf("event = %v, want response", got)
	}
	if got := obj["url"]; got != "https://www.smbc-card.com/index.jsp" {
		t.Fatalf("url = %v", got)
	}
	if got, _ := obj["status"].(float64); int(got) != 403 {
		t.Fatalf("status = %v, want 403", obj["status"])
	}
	if got := obj["content_type"]; got != "text/html" {
		t.Fatalf("content_type = %v, want text/html", got)
	}
	if got, _ := obj["body_size"].(float64); int(got) != 1234 {
		t.Fatalf("body_size = %v, want 1234", obj["body_size"])
	}
	if got := obj["body_preview"]; got != "<html><body>Access Denied</body></html>" {
		t.Fatalf("body_preview = %v", got)
	}
}

func TestFormatStepLine(t *testing.T) {
	line := formatStepLine("2026-06-05T00:00:00Z", "Goto", []string{"https://www.smbc-card.com/mem/index.jsp"})
	var obj map[string]any
	if err := json.Unmarshal([]byte(line), &obj); err != nil {
		t.Fatalf("Unmarshal: %v; line=%q", err, line)
	}
	if got := obj["event"]; got != "step" {
		t.Fatalf("event = %v, want step", got)
	}
	if got := obj["step"]; got != "Goto" {
		t.Fatalf("step = %v, want Goto", got)
	}
	args, ok := obj["args"].([]any)
	if !ok {
		t.Fatalf("args is not a list: %T", obj["args"])
	}
	if len(args) != 1 || args[0] != "https://www.smbc-card.com/mem/index.jsp" {
		t.Fatalf("args = %v", args)
	}
}

func TestFormatStepLineWithNilArgs(t *testing.T) {
	line := formatStepLine("2026-06-05T00:00:00Z", "WaitForURL", nil)
	var obj map[string]any
	if err := json.Unmarshal([]byte(line), &obj); err != nil {
		t.Fatalf("Unmarshal: %v; line=%q", err, line)
	}
	args, ok := obj["args"].([]any)
	if !ok {
		t.Fatalf("args is not a list: %T", obj["args"])
	}
	if len(args) != 0 {
		t.Fatalf("args = %v, want []", args)
	}
}

func TestFormatErrorLine(t *testing.T) {
	line := formatErrorLine(
		"2026-06-05T00:00:00Z",
		"fill VpassID: timeout",
		"https://www.smbc-card.com/mem/index.jsp",
		"<html>Access Denied</html>",
	)
	var obj map[string]any
	if err := json.Unmarshal([]byte(line), &obj); err != nil {
		t.Fatalf("Unmarshal: %v; line=%q", err, line)
	}
	if got := obj["event"]; got != "error" {
		t.Fatalf("event = %v, want error", got)
	}
	if got := obj["error"]; got != "fill VpassID: timeout" {
		t.Fatalf("error = %v", got)
	}
	if got := obj["last_url"]; got != "https://www.smbc-card.com/mem/index.jsp" {
		t.Fatalf("last_url = %v", got)
	}
}

func TestBodyPreview(t *testing.T) {
	cases := []struct {
		name           string
		body           []byte
		wantPreviewLen int
		wantSize       int
	}{
		{name: "empty body", body: nil, wantPreviewLen: 0, wantSize: 0},
		{name: "small body", body: []byte("hi"), wantPreviewLen: 2, wantSize: 2},
		{name: "exactly cap", body: []byte(strings.Repeat("a", maxBodyPreview)), wantPreviewLen: maxBodyPreview, wantSize: maxBodyPreview},
		{name: "larger than cap is truncated", body: []byte(strings.Repeat("a", maxBodyPreview+100)), wantPreviewLen: maxBodyPreview, wantSize: maxBodyPreview + 100},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			preview, size := bodyPreview(tc.body)
			if size != tc.wantSize {
				t.Errorf("size = %d, want %d", size, tc.wantSize)
			}
			if len(preview) != tc.wantPreviewLen {
				t.Errorf("preview length = %d, want %d", len(preview), tc.wantPreviewLen)
			}
		})
	}
}

func TestContentType(t *testing.T) {
	cases := []struct {
		name    string
		headers map[string]string
		want    string
	}{
		{name: "nil headers", headers: nil, want: ""},
		{name: "missing", headers: map[string]string{"server": "Akamai"}, want: ""},
		{name: "present", headers: map[string]string{"content-type": "text/html; charset=utf-8"}, want: "text/html; charset=utf-8"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := contentType(tc.headers); got != tc.want {
				t.Fatalf("contentType() = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestOpenDebugTrace(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("MYSCRAPER_DEBUG_DIR", dir)

	trace := openDebugTrace()
	if trace == nil {
		t.Fatal("openDebugTrace() returned nil with MYSCRAPER_DEBUG_DIR set")
	}
	t.Cleanup(func() { trace.close() })

	trace.appendLine(`{"event":"probe"}`)
	trace.appendLine(`{"event":"probe2"}`)

	got, err := os.ReadFile(filepath.Join(dir, debugTraceFilename))
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	lines := strings.Split(strings.TrimRight(string(got), "\n"), "\n")
	if len(lines) != 2 {
		t.Fatalf("got %d lines, want 2: %q", len(lines), string(got))
	}
	if lines[0] != `{"event":"probe"}` {
		t.Fatalf("lines[0] = %q", lines[0])
	}
	if lines[1] != `{"event":"probe2"}` {
		t.Fatalf("lines[1] = %q", lines[1])
	}
}

func TestOpenDebugTraceDisabled(t *testing.T) {
	t.Setenv("MYSCRAPER_DEBUG_DIR", "")
	if got := openDebugTrace(); got != nil {
		t.Fatalf("openDebugTrace() with empty env = %v, want nil", got)
	}
	t.Setenv("MYSCRAPER_DEBUG_DIR", "   ")
	if got := openDebugTrace(); got != nil {
		t.Fatalf("openDebugTrace() with whitespace env = %v, want nil", got)
	}
}
