package browser

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/playwright-community/playwright-go"
)

const (
	// maxBodyPreview is the maximum number of body bytes captured
	// per response in the trace. 512 is enough to see the start of
	// an HTML page, an Akamai block message, or a JSON error
	// payload, while keeping the trace file small even for sites
	// that serve many small responses.
	maxBodyPreview = 512

	// bodyCaptureTimeout caps how long the trace listener will wait
	// for resp.Body() to return. Playwright's Body() is a synchronous
	// CDP call that blocks until the body is fully received, so a
	// server that opens the connection and never finishes the body
	// (the Akamai / Vpass pattern) would otherwise freeze the
	// listener goroutine and, with it, the entire CDP event loop.
	bodyCaptureTimeout = 1500 * time.Millisecond

	// debugTraceFilename is the file name written into the directory
	// named by MYSCRAPER_DEBUG_DIR. The extension is jsonl so the
	// file is line-oriented and tail-friendly.
	debugTraceFilename = "trace.jsonl"
)

// debugTrace is the per-request recorder for the JSONL trace. It
// owns the open file and remembers the last response URL and body
// preview so an error line can include them as context. The methods
// are safe to call from the playwright "response" goroutine and
// from the workflow goroutine simultaneously; a single mutex
// serialises writes and the snapshot fields.
type debugTrace struct {
	mu              sync.Mutex
	file            *os.File
	lastURL         string
	lastBodyPreview string
}

// openDebugTrace starts a trace in the directory named by
// MYSCRAPER_DEBUG_DIR. It returns nil when the variable is unset or
// whitespace-only, so callers can treat "no trace" as the default
// and skip the listener registration entirely. Failures to create
// the directory or file are reported on stderr and treated as
// "trace disabled" — the trace is best-effort and must never abort
// the actual run.
func openDebugTrace() *debugTrace {
	dir := os.Getenv("MYSCRAPER_DEBUG_DIR")
	if strings.TrimSpace(dir) == "" {
		return nil
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		fmt.Fprintf(os.Stderr, "myscraper: open debug trace dir %q: %v\n", dir, err)
		return nil
	}
	f, err := os.Create(filepath.Join(dir, debugTraceFilename))
	if err != nil {
		fmt.Fprintf(os.Stderr, "myscraper: create trace file: %v\n", err)
		return nil
	}
	return &debugTrace{file: f}
}

func (d *debugTrace) close() {
	if d == nil || d.file == nil {
		return
	}
	_ = d.file.Close()
}

// recordResponse captures one HTTP response served to the page. It
// is called from the playwright "response" event handler, which
// runs in the same goroutine that processes every other CDP event,
// so the body-capture work is delegated to a goroutine and capped
// with bodyCaptureTimeout; otherwise a server that opens a
// connection and never finishes the body would freeze the entire
// event loop.
func (d *debugTrace) recordResponse(resp playwright.Response) {
	if d == nil || resp == nil {
		return
	}
	headers, _ := resp.AllHeaders()
	url := resp.URL()
	status := resp.Status()

	d.mu.Lock()
	d.lastURL = url
	d.lastBodyPreview = "" // refreshed later if the body goroutine wins
	d.mu.Unlock()

	d.appendLine(formatResponseLine(debugNow(), url, status, contentType(headers), 0, ""))
	go d.captureBody(url, resp)
}

// captureBody reads resp.Body() in a goroutine, with a timeout. If
// the body arrives in time, a "body" line is appended to the trace
// and the lastBodyPreview snapshot is updated so a later error line
// can include it. If the body does not arrive, nothing is written
// beyond the "response" line that recordResponse already emitted.
func (d *debugTrace) captureBody(url string, resp playwright.Response) {
	if d == nil || resp == nil {
		return
	}
	bodyCh := make(chan []byte, 1)
	go func() {
		body, _ := resp.Body()
		bodyCh <- body
	}()
	select {
	case body := <-bodyCh:
		preview, size := bodyPreview(body)
		d.mu.Lock()
		d.lastBodyPreview = preview
		d.mu.Unlock()
		d.appendLine(formatBodyLine(debugNow(), url, size, preview))
	case <-time.After(bodyCaptureTimeout):
		// Body never arrived in time; the server likely closed
		// the connection. Leave the response line as-is and move on.
	}
}

// recordStep captures the start of one workflow step. The caller is
// responsible for keeping credentials out of args; the adapter only
// forwards action names and non-sensitive selectors.
func (d *debugTrace) recordStep(step string, args ...string) {
	if d == nil {
		return
	}
	if args == nil {
		args = []string{}
	}
	line := formatStepLine(debugNow(), step, args)
	d.appendLine(line)
}

// recordError captures the error that ended the workflow, together
// with the last seen URL and body preview. Callers should invoke
// this exactly once at the end of a failed run.
func (d *debugTrace) recordError(err error) {
	if d == nil || err == nil {
		return
	}
	d.mu.Lock()
	lastURL := d.lastURL
	lastPreview := d.lastBodyPreview
	d.mu.Unlock()
	line := formatErrorLine(debugNow(), err.Error(), lastURL, lastPreview)
	d.appendLine(line)
}

func (d *debugTrace) appendLine(line string) {
	d.mu.Lock()
	defer d.mu.Unlock()
	if _, err := d.file.WriteString(line + "\n"); err != nil {
		return
	}
	_ = d.file.Sync()
}

func debugNow() string {
	return time.Now().UTC().Format(time.RFC3339Nano)
}

// bodyPreview returns the first maxBodyPreview bytes of body as a
// string and the total body size. Non-UTF8 bytes are preserved by
// the JSON encoder as escaped sequences.
func bodyPreview(body []byte) (string, int) {
	if len(body) == 0 {
		return "", 0
	}
	if len(body) <= maxBodyPreview {
		return string(body), len(body)
	}
	return string(body[:maxBodyPreview]), len(body)
}

func contentType(headers map[string]string) string {
	if headers == nil {
		return ""
	}
	if v, ok := headers["content-type"]; ok {
		return v
	}
	return ""
}

// formatResponseLine returns a JSONL line describing one HTTP response.
// It is exported-via-lowercase-name for tests and is intentionally
// pure so it can be exercised without a real browser.
func formatResponseLine(ts, url string, status int, contentType string, bodySize int, bodyPreview string) string {
	return marshalLine(map[string]any{
		"ts":           ts,
		"event":        "response",
		"url":          url,
		"status":       status,
		"content_type": contentType,
		"body_size":    bodySize,
		"body_preview": bodyPreview,
	})
}

// formatBodyLine returns a JSONL line carrying the body preview
// captured after the initial "response" line. It shares the URL
// with the response line so a reader can correlate the two.
func formatBodyLine(ts, url string, bodySize int, bodyPreview string) string {
	return marshalLine(map[string]any{
		"ts":           ts,
		"event":        "body",
		"url":          url,
		"body_size":    bodySize,
		"body_preview": bodyPreview,
	})
}

// formatStepLine returns a JSONL line describing the start of a
// workflow step. The args slice is included verbatim and must not
// contain credentials.
func formatStepLine(ts, step string, args []string) string {
	if args == nil {
		args = []string{}
	}
	return marshalLine(map[string]any{
		"ts":    ts,
		"event": "step",
		"step":  step,
		"args":  args,
	})
}

// formatErrorLine returns a JSONL line describing the error that
// ended the workflow, with the last seen URL and body preview as
// context.
func formatErrorLine(ts, errMsg, lastURL, lastBodyPreview string) string {
	return marshalLine(map[string]any{
		"ts":                ts,
		"event":             "error",
		"error":             errMsg,
		"last_url":          lastURL,
		"last_body_preview": lastBodyPreview,
	})
}

func marshalLine(obj map[string]any) string {
	b, err := json.Marshal(obj)
	if err != nil {
		// Fall back to a minimal error line so the trace still
		// records the failure of the formatter itself.
		fallback, _ := json.Marshal(map[string]any{
			"ts":    obj["ts"],
			"event": "format_error",
			"error": err.Error(),
		})
		return string(fallback)
	}
	return string(b)
}
