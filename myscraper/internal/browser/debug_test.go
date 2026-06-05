package browser

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/playwright-community/playwright-go"
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

// fakeResponse is a minimal playwright.Response used to exercise the
// body-capture goroutine without a real browser. Only the methods
// the trace listener actually calls are implemented; the rest panic
// so accidental coupling is caught.
type fakeResponse struct {
	url        string
	status     int
	headers    map[string]string
	body       []byte
	bodyDelay  time.Duration
}

func (f fakeResponse) URL() string                                  { return f.url }
func (f fakeResponse) Status() int                                 { return f.status }
func (f fakeResponse) AllHeaders() (map[string]string, error)       { return f.headers, nil }
func (f fakeResponse) Body() ([]byte, error) {
	if f.bodyDelay > 0 {
		time.Sleep(f.bodyDelay)
	}
	return f.body, nil
}

// stub the rest of the playwright.Response surface so the type
// satisfies the interface and the compiler enforces no new method
// dependencies sneak in silently. Unused methods return zero values
// rather than panicking so a future reader does not get a misleading
// "not used" message for a method that turns out to be used.
func (fakeResponse) Headers() map[string]string { return nil }
func (fakeResponse) HeadersArray() ([]playwright.NameValue, error) {
	return nil, nil
}
func (fakeResponse) HeaderValue(string) (string, error)    { return "", nil }
func (fakeResponse) HeaderValues(string) ([]string, error) { return nil, nil }
func (fakeResponse) StatusText() string                    { return "" }
func (fakeResponse) Ok() bool                              { return false }
func (fakeResponse) Text() (string, error)                 { return "", nil }
func (fakeResponse) JSON(any) error                        { return nil }
func (fakeResponse) Finished() error                       { return nil }
func (fakeResponse) Frame() playwright.Frame               { return nil }
func (fakeResponse) Request() playwright.Request           { return nil }
func (fakeResponse) FromServiceWorker() bool               { return false }
func (fakeResponse) SecurityDetails() (*playwright.ResponseSecurityDetailsResult, error) {
	return nil, nil
}
func (fakeResponse) ServerAddr() (*playwright.ResponseServerAddrResult, error) {
	return nil, nil
}

func TestRecordResponseFastBody(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("MYSCRAPER_DEBUG_DIR", dir)

	trace := openDebugTrace()
	if trace == nil {
		t.Fatal("openDebugTrace() returned nil")
	}
	t.Cleanup(func() { trace.close() })

	trace.recordResponse(fakeResponse{
		url:     "https://www.smbc-card.com/index.jsp",
		status:  403,
		headers: map[string]string{"content-type": "text/html"},
		body:    []byte("<html>Access Denied</html>"),
	})

	// Wait briefly for the body goroutine to write its line.
	deadline := time.Now().Add(time.Second)
	for time.Now().Before(deadline) {
		got, _ := os.ReadFile(filepath.Join(dir, debugTraceFilename))
		if strings.Count(string(got), "\n") >= 2 {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}

	got, err := os.ReadFile(filepath.Join(dir, debugTraceFilename))
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	lines := strings.Split(strings.TrimRight(string(got), "\n"), "\n")
	if len(lines) != 2 {
		t.Fatalf("got %d lines, want 2: %q", len(lines), string(got))
	}
	var first, second map[string]any
	_ = json.Unmarshal([]byte(lines[0]), &first)
	_ = json.Unmarshal([]byte(lines[1]), &second)
	if first["event"] != "response" {
		t.Fatalf("lines[0].event = %v, want response", first["event"])
	}
	if got, _ := first["status"].(float64); int(got) != 403 {
		t.Fatalf("lines[0].status = %v, want 403", first["status"])
	}
	if first["body_preview"] != "" {
		t.Fatalf("lines[0].body_preview = %v, want empty (body captured separately)", first["body_preview"])
	}
	if second["event"] != "body" {
		t.Fatalf("lines[1].event = %v, want body", second["event"])
	}
	if second["url"] != "https://www.smbc-card.com/index.jsp" {
		t.Fatalf("lines[1].url = %v", second["url"])
	}
	if second["body_preview"] != "<html>Access Denied</html>" {
		t.Fatalf("lines[1].body_preview = %v", second["body_preview"])
	}
}

func TestRecordResponseSlowBodyDoesNotBlockListener(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("MYSCRAPER_DEBUG_DIR", dir)

	trace := openDebugTrace()
	if trace == nil {
		t.Fatal("openDebugTrace() returned nil")
	}
	t.Cleanup(func() { trace.close() })

	// Body that takes much longer than the listener should wait.
	// If the listener blocks on Body(), the assertion below will
	// time out and the test will fail.
	done := make(chan struct{})
	go func() {
		defer close(done)
		trace.recordResponse(fakeResponse{
			url:       "https://www.smbc-card.com/index.jsp",
			status:    200,
			headers:   map[string]string{"content-type": "text/html"},
			bodyDelay: 5 * time.Second,
			body:      []byte("<html>ok</html>"),
		})
	}()
	select {
	case <-done:
	case <-time.After(500 * time.Millisecond):
		t.Fatal("recordResponse blocked on Body(); the listener must return immediately")
	}

	// The response line is on disk; the body line is not (and
	// will not be, within the body capture timeout).
	got, _ := os.ReadFile(filepath.Join(dir, debugTraceFilename))
	if !strings.Contains(string(got), `"event":"response"`) {
		t.Fatalf("trace missing response line: %q", string(got))
	}
	if strings.Contains(string(got), `"event":"body"`) {
		t.Fatalf("trace should not contain body line yet, got: %q", string(got))
	}
}

func TestFormatBodyLine(t *testing.T) {
	line := formatBodyLine("2026-06-05T00:00:00Z", "https://example.com", 42, "hello")
	var obj map[string]any
	if err := json.Unmarshal([]byte(line), &obj); err != nil {
		t.Fatalf("Unmarshal: %v; line=%q", err, line)
	}
	if obj["event"] != "body" {
		t.Fatalf("event = %v, want body", obj["event"])
	}
	if obj["url"] != "https://example.com" {
		t.Fatalf("url = %v", obj["url"])
	}
	if got, _ := obj["body_size"].(float64); int(got) != 42 {
		t.Fatalf("body_size = %v, want 42", obj["body_size"])
	}
	if obj["body_preview"] != "hello" {
		t.Fatalf("body_preview = %v", obj["body_preview"])
	}
}
