# Myscraper SMBC Card Web Meisai HTML Dump Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add a first authenticated SMBC Card flow to `myscraper` that logs in with Vpass, opens the WEB明細 page, and saves the resulting HTML snapshot for later CSV parsing work.

**Architecture:** Keep the current anonymous `--url` fetch flow intact and add one explicit authenticated mode for `smbc-card-webmeisai`. Put SMBC-specific constants, credential loading, and navigation logic in `internal/smbccard`, let `internal/scrape` switch between generic fetch and the new authenticated flow, and keep `internal/browser` responsible only for Playwright lifecycle plus adapting Playwright pages to the SMBC workflow interface. HTML dump is the terminal output for this increment; CSV conversion is intentionally out of scope for this plan.

**Tech Stack:** Go 1.24, `github.com/playwright-community/playwright-go` v0.5200.1, Nix dev shell, Playwright Chromium, Go standard library

---

## Planned File Structure

- Modify: `README.md`
- Modify: `myscraper/internal/cli/options.go`
- Modify: `myscraper/internal/cli/options_test.go`
- Modify: `myscraper/internal/cli/run.go`
- Modify: `myscraper/internal/cli/run_test.go`
- Create: `myscraper/internal/smbccard/credentials.go`
- Create: `myscraper/internal/smbccard/credentials_test.go`
- Create: `myscraper/internal/smbccard/automation.go`
- Create: `myscraper/internal/smbccard/automation_test.go`
- Modify: `myscraper/internal/scrape/service.go`
- Modify: `myscraper/internal/scrape/service_test.go`
- Modify: `myscraper/internal/browser/playwright.go`
- Create: `myscraper/e2e/smbccard_webmeisai_smoke_test.go`

`internal/smbccard` is the new seam for this work. It keeps selectors, URLs, and env var rules out of the generic scrape service so CSV parsing can be added later without bloating `internal/browser` or `internal/cli`.

### Task 1: Extend CLI Options For Authenticated SMBC Mode

**Files:**
- Modify: `myscraper/internal/cli/options.go`
- Modify: `myscraper/internal/cli/options_test.go`

- [ ] **Step 1: Write the failing test**

`myscraper/internal/cli/options_test.go`

```go
package cli

import "testing"

func TestParseArgs(t *testing.T) {
	t.Run("requires url for fetch mode", func(t *testing.T) {
		_, err := ParseArgs([]string{"--mode", ModeFetchURL})
		if err == nil || err.Error() != "--url is required when --mode=fetch-url" {
			t.Fatalf("expected url validation error, got %v", err)
		}
	})

	t.Run("applies fetch defaults", func(t *testing.T) {
		got, err := ParseArgs([]string{"--url", "https://github.com"})
		if err != nil {
			t.Fatalf("ParseArgs() error = %v", err)
		}
		if got.Mode != ModeFetchURL {
			t.Fatalf("Mode = %q, want %q", got.Mode, ModeFetchURL)
		}
		if got.OutputPath != "tmp/page.html" {
			t.Fatalf("OutputPath = %q, want %q", got.OutputPath, "tmp/page.html")
		}
	})

	t.Run("applies smbc defaults without url", func(t *testing.T) {
		got, err := ParseArgs([]string{"--mode", "smbc-card-webmeisai"})
		if err != nil {
			t.Fatalf("ParseArgs() error = %v", err)
		}
		if got.Mode != "smbc-card-webmeisai" {
			t.Fatalf("Mode = %q, want %q", got.Mode, "smbc-card-webmeisai")
		}
		if got.URL != "" {
			t.Fatalf("URL = %q, want empty", got.URL)
		}
		if got.OutputPath != "tmp/smbccard-webmeisai.html" {
			t.Fatalf("OutputPath = %q, want %q", got.OutputPath, "tmp/smbccard-webmeisai.html")
		}
	})

	t.Run("rejects unknown mode", func(t *testing.T) {
		_, err := ParseArgs([]string{"--mode", "unknown"})
		if err == nil || err.Error() != "unsupported --mode: unknown" {
			t.Fatalf("expected mode validation error, got %v", err)
		}
	})
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `nix develop -c bash -lc 'cd myscraper && go test ./internal/cli -run TestParseArgs -v'`
Expected: FAIL because `ModeFetchURL` is undefined and `ParseArgs` still requires `--url` unconditionally

- [ ] **Step 3: Write minimal implementation**

`myscraper/internal/cli/options.go`

```go
package cli

import (
	"errors"
	"flag"
	"io"

	"github.com/azuki774/myscrapers/myscraper/internal/smbccard"
)

const ModeFetchURL = "fetch-url"

type Options struct {
	Mode       string
	URL        string
	OutputPath string
	Headless   bool
}

func ParseArgs(args []string) (Options, error) {
	fs := flag.NewFlagSet("myscraper", flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	opts := Options{}
	fs.StringVar(&opts.Mode, "mode", ModeFetchURL, "run mode")
	fs.StringVar(&opts.URL, "url", "", "target URL to scrape when --mode=fetch-url")
	fs.StringVar(&opts.OutputPath, "out", "", "path to write HTML snapshot")
	fs.BoolVar(&opts.Headless, "headless", true, "launch browser in headless mode")

	if err := fs.Parse(args); err != nil {
		return Options{}, err
	}

	switch opts.Mode {
	case ModeFetchURL:
		if opts.URL == "" {
			return Options{}, errors.New("--url is required when --mode=fetch-url")
		}
		if opts.OutputPath == "" {
			opts.OutputPath = "tmp/page.html"
		}
	case smbccard.ModeWebMeisaiHTMLDump:
		if opts.OutputPath == "" {
			opts.OutputPath = "tmp/smbccard-webmeisai.html"
		}
	default:
		return Options{}, errors.New("unsupported --mode: " + opts.Mode)
	}

	return opts, nil
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `nix develop -c bash -lc 'cd myscraper && go test ./internal/cli -run TestParseArgs -v'`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add myscraper/internal/cli/options.go myscraper/internal/cli/options_test.go
git commit -m "feat: add myscraper smbc mode selection"
```

### Task 2: Add SMBC Credentials And Workflow Orchestration

**Files:**
- Create: `myscraper/internal/smbccard/credentials.go`
- Create: `myscraper/internal/smbccard/credentials_test.go`
- Create: `myscraper/internal/smbccard/automation.go`
- Create: `myscraper/internal/smbccard/automation_test.go`

- [ ] **Step 1: Write the failing tests**

`myscraper/internal/smbccard/credentials_test.go`

```go
package smbccard

import (
	"os"
	"testing"
)

func TestLoadCredentialsFromEnv(t *testing.T) {
	t.Run("prefers dedicated env vars", func(t *testing.T) {
		t.Setenv("SMBC_VPASS_ID", "dedicated-id")
		t.Setenv("SMBC_VPASS_PASSWORD", "dedicated-pass")
		t.Setenv("user", "legacy-id")
		t.Setenv("pass", "legacy-pass")

		got, err := LoadCredentialsFromEnv()
		if err != nil {
			t.Fatalf("LoadCredentialsFromEnv() error = %v", err)
		}
		if got.LoginID != "dedicated-id" || got.Password != "dedicated-pass" {
			t.Fatalf("credentials = %#v", got)
		}
	})

	t.Run("falls back to legacy env vars", func(t *testing.T) {
		t.Setenv("SMBC_VPASS_ID", "")
		t.Setenv("SMBC_VPASS_PASSWORD", "")
		t.Setenv("user", "legacy-id")
		t.Setenv("pass", "legacy-pass")

		got, err := LoadCredentialsFromEnv()
		if err != nil {
			t.Fatalf("LoadCredentialsFromEnv() error = %v", err)
		}
		if got.LoginID != "legacy-id" || got.Password != "legacy-pass" {
			t.Fatalf("credentials = %#v", got)
		}
	})

	t.Run("rejects missing credentials", func(t *testing.T) {
		t.Setenv("SMBC_VPASS_ID", "")
		t.Setenv("SMBC_VPASS_PASSWORD", "")
		t.Setenv("user", "")
		t.Setenv("pass", "")

		_, err := LoadCredentialsFromEnv()
		if err == nil || err.Error() != "SMBC credentials are required: set SMBC_VPASS_ID/SMBC_VPASS_PASSWORD or user/pass" {
			t.Fatalf("expected missing credentials error, got %v", err)
		}
	})
}

func TestEnvOrFallback(t *testing.T) {
	t.Setenv("PRIMARY", "")
	t.Setenv("FALLBACK", "fallback-value")

	if got := envOrFallback("PRIMARY", "FALLBACK"); got != "fallback-value" {
		t.Fatalf("envOrFallback() = %q, want %q", got, "fallback-value")
	}

	if _, ok := os.LookupEnv("PRIMARY"); !ok {
		t.Fatalf("PRIMARY should still exist in environment during the test")
	}
}
```

`myscraper/internal/smbccard/automation_test.go`

```go
package smbccard

import (
	"errors"
	"reflect"
	"testing"
)

type fakePage struct {
	steps   []string
	title   string
	html    string
	url     string
	waitErr error
}

func (p *fakePage) Goto(url string) error {
	p.steps = append(p.steps, "goto:"+url)
	p.url = url
	return nil
}

func (p *fakePage) FillByLabel(label, value string) error {
	p.steps = append(p.steps, "fill:"+label+"="+value)
	return nil
}

func (p *fakePage) ClickButton(name string) error {
	p.steps = append(p.steps, "click:"+name)
	return nil
}

func (p *fakePage) WaitForURL(pattern string) error {
	p.steps = append(p.steps, "wait:"+pattern)
	if p.waitErr != nil {
		return p.waitErr
	}
	if pattern == MyPageURLPattern {
		p.url = MyPageURL
	}
	if pattern == WebMeisaiURLPattern {
		p.url = WebMeisaiURL
	}
	return nil
}

func (p *fakePage) Title() (string, error) {
	return p.title, nil
}

func (p *fakePage) Content() (string, error) {
	return p.html, nil
}

func (p *fakePage) URL() string {
	return p.url
}

func TestCaptureWebMeisai(t *testing.T) {
	page := &fakePage{
		title: "WEB明細",
		html:  "<html><body>statement</body></html>",
	}

	got, err := CaptureWebMeisai(page, Credentials{
		LoginID:  "member-id",
		Password: "member-pass",
	})
	if err != nil {
		t.Fatalf("CaptureWebMeisai() error = %v", err)
	}

	wantSteps := []string{
		"goto:" + TopURL,
		"goto:" + LoginURL,
		"fill:VpassID=member-id",
		"fill:パスワード=member-pass",
		"click:ログイン",
		"wait:" + MyPageURLPattern,
		"goto:" + WebMeisaiURL,
		"wait:" + WebMeisaiURLPattern,
	}
	if !reflect.DeepEqual(page.steps, wantSteps) {
		t.Fatalf("steps = %#v, want %#v", page.steps, wantSteps)
	}
	if got.URL != WebMeisaiURL {
		t.Fatalf("URL = %q, want %q", got.URL, WebMeisaiURL)
	}
	if got.Title != "WEB明細" {
		t.Fatalf("Title = %q, want %q", got.Title, "WEB明細")
	}
	if got.HTML != "<html><body>statement</body></html>" {
		t.Fatalf("HTML = %q", got.HTML)
	}
}

func TestCaptureWebMeisaiReturnsWaitError(t *testing.T) {
	page := &fakePage{waitErr: errors.New("wait failed")}

	_, err := CaptureWebMeisai(page, Credentials{
		LoginID:  "member-id",
		Password: "member-pass",
	})
	if err == nil || err.Error() != "wait for mypage: wait failed" {
		t.Fatalf("expected wait error, got %v", err)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `nix develop -c bash -lc 'cd myscraper && go test ./internal/smbccard -v'`
Expected: FAIL with `no Go files` or `undefined` errors for `LoadCredentialsFromEnv`, `CaptureWebMeisai`, and the SMBC URL constants

- [ ] **Step 3: Write minimal implementation**

`myscraper/internal/smbccard/credentials.go`

```go
package smbccard

import (
	"errors"
	"os"
)

const ModeWebMeisaiHTMLDump = "smbc-card-webmeisai"

type Credentials struct {
	LoginID  string
	Password string
}

func LoadCredentialsFromEnv() (Credentials, error) {
	loginID := envOrFallback("SMBC_VPASS_ID", "user")
	password := envOrFallback("SMBC_VPASS_PASSWORD", "pass")
	if loginID == "" || password == "" {
		return Credentials{}, errors.New("SMBC credentials are required: set SMBC_VPASS_ID/SMBC_VPASS_PASSWORD or user/pass")
	}
	return Credentials{
		LoginID:  loginID,
		Password: password,
	}, nil
}

func envOrFallback(primary, fallback string) string {
	if value := os.Getenv(primary); value != "" {
		return value
	}
	return os.Getenv(fallback)
}
```

`myscraper/internal/smbccard/automation.go`

```go
package smbccard

import (
	"fmt"

	"github.com/azuki774/myscrapers/myscraper/internal/scrape"
)

const (
	TopURL              = "https://www.smbc-card.com/index.jsp"
	LoginURL            = "https://www.smbc-card.com/mem/index.jsp"
	MyPageURL           = "https://www.smbc-card.com/memx/mypage/index.html"
	MyPageURLPattern    = "**/memx/mypage/index.html"
	WebMeisaiURL        = "https://www.smbc-card.com/memx/web_meisai/top/index.html#info2"
	WebMeisaiURLPattern = "**/memx/web_meisai/top/index.html*"
)

type InteractivePage interface {
	Goto(url string) error
	FillByLabel(label, value string) error
	ClickButton(name string) error
	WaitForURL(pattern string) error
	Title() (string, error)
	Content() (string, error)
	URL() string
}

func CaptureWebMeisai(page InteractivePage, creds Credentials) (scrape.PageSnapshot, error) {
	if err := page.Goto(TopURL); err != nil {
		return scrape.PageSnapshot{}, fmt.Errorf("goto top page: %w", err)
	}
	if err := page.Goto(LoginURL); err != nil {
		return scrape.PageSnapshot{}, fmt.Errorf("goto login page: %w", err)
	}
	if err := page.FillByLabel("VpassID", creds.LoginID); err != nil {
		return scrape.PageSnapshot{}, fmt.Errorf("fill VpassID: %w", err)
	}
	if err := page.FillByLabel("パスワード", creds.Password); err != nil {
		return scrape.PageSnapshot{}, fmt.Errorf("fill password: %w", err)
	}
	if err := page.ClickButton("ログイン"); err != nil {
		return scrape.PageSnapshot{}, fmt.Errorf("click login: %w", err)
	}
	if err := page.WaitForURL(MyPageURLPattern); err != nil {
		return scrape.PageSnapshot{}, fmt.Errorf("wait for mypage: %w", err)
	}
	if err := page.Goto(WebMeisaiURL); err != nil {
		return scrape.PageSnapshot{}, fmt.Errorf("goto web meisai page: %w", err)
	}
	if err := page.WaitForURL(WebMeisaiURLPattern); err != nil {
		return scrape.PageSnapshot{}, fmt.Errorf("wait for web meisai page: %w", err)
	}

	title, err := page.Title()
	if err != nil {
		return scrape.PageSnapshot{}, fmt.Errorf("read title: %w", err)
	}
	html, err := page.Content()
	if err != nil {
		return scrape.PageSnapshot{}, fmt.Errorf("read content: %w", err)
	}

	return scrape.PageSnapshot{
		URL:   page.URL(),
		Title: title,
		HTML:  html,
	}, nil
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `nix develop -c bash -lc 'cd myscraper && go test ./internal/smbccard -v'`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add myscraper/internal/smbccard/credentials.go myscraper/internal/smbccard/credentials_test.go myscraper/internal/smbccard/automation.go myscraper/internal/smbccard/automation_test.go
git commit -m "feat: add smbc card workflow package"
```

### Task 3: Route The New Mode Through The CLI, Scrape Service, And Playwright

**Files:**
- Modify: `myscraper/internal/cli/run.go`
- Modify: `myscraper/internal/cli/run_test.go`
- Modify: `myscraper/internal/scrape/service.go`
- Modify: `myscraper/internal/scrape/service_test.go`
- Modify: `myscraper/internal/browser/playwright.go`

- [ ] **Step 1: Write the failing tests**

`myscraper/internal/cli/run_test.go`

```go
package cli

import (
	"bytes"
	"context"
	"testing"

	"github.com/azuki774/myscrapers/myscraper/internal/scrape"
)

type fakeRunner struct {
	result scrape.Result
	err    error
	req    scrape.Request
}

func (f *fakeRunner) Run(ctx context.Context, req scrape.Request) (scrape.Result, error) {
	f.req = req
	return f.result, f.err
}

func TestRunPrintsSavedPath(t *testing.T) {
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	runner := &fakeRunner{
		result: scrape.Result{
			Title:      "WEB明細",
			OutputPath: "tmp/smbc.html",
		},
	}

	exitCode := Run(
		[]string{"--mode", "smbc-card-webmeisai", "--out", "tmp/smbc.html"},
		stdout,
		stderr,
		runner,
	)

	if exitCode != 0 {
		t.Fatalf("exitCode = %d, want 0; stderr = %q", exitCode, stderr.String())
	}
	if runner.req.Mode != "smbc-card-webmeisai" {
		t.Fatalf("Mode = %q, want %q", runner.req.Mode, "smbc-card-webmeisai")
	}
	if stdout.String() != "saved tmp/smbc.html (WEB明細)
" {
		t.Fatalf("stdout = %q", stdout.String())
	}
}
```

`myscraper/internal/scrape/service_test.go`

```go
package scrape

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/azuki774/myscrapers/myscraper/internal/smbccard"
)

type fakeBrowser struct {
	fetchSnapshot   PageSnapshot
	smbcSnapshot    PageSnapshot
	err             error
	fetchCalledWith string
	smbcCalledWith  smbccard.Credentials
}

func (f *fakeBrowser) Fetch(ctx context.Context, url string, headless bool) (PageSnapshot, error) {
	f.fetchCalledWith = url
	return f.fetchSnapshot, f.err
}

func (f *fakeBrowser) FetchSMBCCardWebMeisai(ctx context.Context, creds smbccard.Credentials, headless bool) (PageSnapshot, error) {
	f.smbcCalledWith = creds
	return f.smbcSnapshot, f.err
}

func TestServiceRunFetchModeWritesHTML(t *testing.T) {
	out := filepath.Join(t.TempDir(), "page.html")
	browser := &fakeBrowser{
		fetchSnapshot: PageSnapshot{
			URL:   "https://github.com",
			Title: "GitHub",
			HTML:  "<html><body><h1>GitHub</h1></body></html>",
		},
	}
	svc := Service{Browser: browser}

	result, err := svc.Run(context.Background(), Request{
		Mode:       "fetch-url",
		URL:        "https://github.com",
		OutputPath: out,
		Headless:   true,
	})
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if result.Title != "GitHub" {
		t.Fatalf("Title = %q, want %q", result.Title, "GitHub")
	}
	if browser.fetchCalledWith != "https://github.com" {
		t.Fatalf("fetchCalledWith = %q", browser.fetchCalledWith)
	}
}

func TestServiceRunSMBCModeWritesHTML(t *testing.T) {
	t.Setenv("SMBC_VPASS_ID", "member-id")
	t.Setenv("SMBC_VPASS_PASSWORD", "member-pass")

	out := filepath.Join(t.TempDir(), "smbc.html")
	browser := &fakeBrowser{
		smbcSnapshot: PageSnapshot{
			URL:   smbccard.WebMeisaiURL,
			Title: "WEB明細",
			HTML:  "<html><body>statement</body></html>",
		},
	}
	svc := Service{Browser: browser}

	result, err := svc.Run(context.Background(), Request{
		Mode:       smbccard.ModeWebMeisaiHTMLDump,
		OutputPath: out,
		Headless:   true,
	})
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if result.Title != "WEB明細" {
		t.Fatalf("Title = %q, want %q", result.Title, "WEB明細")
	}
	if browser.smbcCalledWith.LoginID != "member-id" || browser.smbcCalledWith.Password != "member-pass" {
		t.Fatalf("smbcCalledWith = %#v", browser.smbcCalledWith)
	}

	data, err := os.ReadFile(out)
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}
	if string(data) != "<html><body>statement</body></html>" {
		t.Fatalf("HTML = %q", string(data))
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `nix develop -c bash -lc 'cd myscraper && go test ./internal/cli ./internal/scrape -run TestRun -v'`
Expected: FAIL because `Request` does not have a `Mode` field, `run.go` does not forward `opts.Mode`, and `Browser` does not have `FetchSMBCCardWebMeisai`

- [ ] **Step 3: Write minimal implementation**

`myscraper/internal/cli/run.go`

```go
package cli

import (
	"context"
	"fmt"
	"io"

	"github.com/azuki774/myscrapers/myscraper/internal/scrape"
)

type Runner interface {
	Run(ctx context.Context, req scrape.Request) (scrape.Result, error)
}

func Run(args []string, stdout, stderr io.Writer, runner Runner) int {
	opts, err := ParseArgs(args)
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 2
	}

	result, err := runner.Run(context.Background(), scrape.Request{
		Mode:       opts.Mode,
		URL:        opts.URL,
		OutputPath: opts.OutputPath,
		Headless:   opts.Headless,
	})
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}

	fmt.Fprintf(stdout, "saved %s (%s)
", result.OutputPath, result.Title)
	return 0
}
```

`myscraper/internal/scrape/service.go`

```go
package scrape

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/azuki774/myscrapers/myscraper/internal/smbccard"
)

type Browser interface {
	Fetch(ctx context.Context, url string, headless bool) (PageSnapshot, error)
	FetchSMBCCardWebMeisai(ctx context.Context, creds smbccard.Credentials, headless bool) (PageSnapshot, error)
}

type PageSnapshot struct {
	URL   string
	Title string
	HTML  string
}

type Request struct {
	Mode       string
	URL        string
	OutputPath string
	Headless   bool
}

type Result struct {
	Title      string
	OutputPath string
}

type Service struct {
	Browser Browser
}

func (s Service) Run(ctx context.Context, req Request) (Result, error) {
	if s.Browser == nil {
		return Result{}, fmt.Errorf("browser is required")
	}

	var (
		snapshot PageSnapshot
		err      error
	)

	switch req.Mode {
	case "", "fetch-url":
		snapshot, err = s.Browser.Fetch(ctx, req.URL, req.Headless)
	case smbccard.ModeWebMeisaiHTMLDump:
		creds, loadErr := smbccard.LoadCredentialsFromEnv()
		if loadErr != nil {
			return Result{}, loadErr
		}
		snapshot, err = s.Browser.FetchSMBCCardWebMeisai(ctx, creds, req.Headless)
	default:
		return Result{}, fmt.Errorf("unsupported scrape mode: %s", req.Mode)
	}
	if err != nil {
		return Result{}, err
	}

	if err := os.MkdirAll(filepath.Dir(req.OutputPath), 0o755); err != nil {
		return Result{}, err
	}
	if err := os.WriteFile(req.OutputPath, []byte(snapshot.HTML), 0o644); err != nil {
		return Result{}, err
	}

	return Result{
		Title:      snapshot.Title,
		OutputPath: req.OutputPath,
	}, nil
}
```

`myscraper/internal/browser/playwright.go`

```go
package browser

import (
	"context"
	"fmt"
	"os"
	"os/exec"

	"github.com/azuki774/myscrapers/myscraper/internal/scrape"
	"github.com/azuki774/myscrapers/myscraper/internal/smbccard"
	"github.com/playwright-community/playwright-go"
)

type PlaywrightBrowser struct{}

type playwrightInteractivePage struct {
	page playwright.Page
}

func InstallDriver() error {
	return playwright.Install(runOptions())
}

func runOptions() *playwright.RunOptions {
	driverDirectory := os.Getenv("PLAYWRIGHT_DRIVER_PATH")
	if driverDirectory == "" {
		driverDirectory = ".playwright-driver"
	}
	return &playwright.RunOptions{
		DriverDirectory:     driverDirectory,
		SkipInstallBrowsers: true,
	}
}

func chromiumExecutablePath() (string, error) {
	for _, candidate := range []string{"chromium", "chromium-browser", "google-chrome", "google-chrome-stable"} {
		if path, err := exec.LookPath(candidate); err == nil {
			return path, nil
		}
	}
	return "", fmt.Errorf("no chromium-compatible browser found in PATH")
}

func (PlaywrightBrowser) Fetch(ctx context.Context, url string, headless bool) (scrape.PageSnapshot, error) {
	return withPage(ctx, headless, func(page playwright.Page) (scrape.PageSnapshot, error) {
		if _, err := page.Goto(url); err != nil {
			return scrape.PageSnapshot{}, fmt.Errorf("goto %s: %w", url, err)
		}
		title, err := page.Title()
		if err != nil {
			return scrape.PageSnapshot{}, fmt.Errorf("read title: %w", err)
		}
		html, err := page.Content()
		if err != nil {
			return scrape.PageSnapshot{}, fmt.Errorf("read content: %w", err)
		}
		return scrape.PageSnapshot{
			URL:   page.URL(),
			Title: title,
			HTML:  html,
		}, nil
	})
}

func (PlaywrightBrowser) FetchSMBCCardWebMeisai(ctx context.Context, creds smbccard.Credentials, headless bool) (scrape.PageSnapshot, error) {
	return withPage(ctx, headless, func(page playwright.Page) (scrape.PageSnapshot, error) {
		return smbccard.CaptureWebMeisai(playwrightInteractivePage{page: page}, creds)
	})
}

func withPage(ctx context.Context, headless bool, run func(page playwright.Page) (scrape.PageSnapshot, error)) (scrape.PageSnapshot, error) {
	if err := InstallDriver(); err != nil {
		return scrape.PageSnapshot{}, fmt.Errorf("install playwright driver: %w", err)
	}

	pw, err := playwright.Run(runOptions())
	if err != nil {
		return scrape.PageSnapshot{}, fmt.Errorf("start playwright: %w", err)
	}
	defer pw.Stop()

	executablePath, err := chromiumExecutablePath()
	if err != nil {
		return scrape.PageSnapshot{}, err
	}

	browser, err := pw.Chromium.Launch(playwright.BrowserTypeLaunchOptions{
		ExecutablePath: playwright.String(executablePath),
		Headless:       playwright.Bool(headless),
	})
	if err != nil {
		return scrape.PageSnapshot{}, fmt.Errorf("launch chromium: %w", err)
	}
	defer browser.Close()

	page, err := browser.NewPage()
	if err != nil {
		return scrape.PageSnapshot{}, fmt.Errorf("new page: %w", err)
	}

	return run(page)
}

func (p playwrightInteractivePage) Goto(url string) error {
	_, err := p.page.Goto(url, playwright.PageGotoOptions{
		WaitUntil: playwright.WaitUntilStateNetworkidle,
	})
	return err
}

func (p playwrightInteractivePage) FillByLabel(label, value string) error {
	return p.page.GetByLabel(label).Fill(value)
}

func (p playwrightInteractivePage) ClickButton(name string) error {
	return p.page.GetByRole(playwright.AriaRoleButton, playwright.PageGetByRoleOptions{
		Name: name,
	}).Click()
}

func (p playwrightInteractivePage) WaitForURL(pattern string) error {
	_, err := p.page.WaitForURL(pattern)
	return err
}

func (p playwrightInteractivePage) Title() (string, error) {
	return p.page.Title()
}

func (p playwrightInteractivePage) Content() (string, error) {
	return p.page.Content()
}

func (p playwrightInteractivePage) URL() string {
	return p.page.URL()
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `nix develop -c bash -lc 'cd myscraper && go test ./internal/cli ./internal/scrape ./internal/browser ./internal/smbccard -v'`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add myscraper/internal/cli/run.go myscraper/internal/cli/run_test.go myscraper/internal/scrape/service.go myscraper/internal/scrape/service_test.go myscraper/internal/browser/playwright.go
git commit -m "feat: add smbc card authenticated html dump flow"
```

### Task 4: Add A Gated Real-Browser Smoke Test And Usage Docs

**Files:**
- Create: `myscraper/e2e/smbccard_webmeisai_smoke_test.go`
- Modify: `README.md`

- [ ] **Step 1: Write the failing test**

`myscraper/e2e/smbccard_webmeisai_smoke_test.go`

```go
package e2e

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/azuki774/myscrapers/myscraper/internal/browser"
	"github.com/azuki774/myscrapers/myscraper/internal/scrape"
	"github.com/azuki774/myscrapers/myscraper/internal/smbccard"
)

func TestSMBCCardWebMeisaiSmoke(t *testing.T) {
	if os.Getenv("PLAYWRIGHT_E2E_SMBCCARD") != "1" {
		t.Skip("set PLAYWRIGHT_E2E_SMBCCARD=1 to run the SMBC Card smoke test")
	}
	if os.Getenv("SMBC_VPASS_ID") == "" && os.Getenv("user") == "" {
		t.Skip("set SMBC_VPASS_ID/SMBC_VPASS_PASSWORD or user/pass to run the SMBC Card smoke test")
	}

	out := filepath.Join(t.TempDir(), "smbccard-webmeisai.html")
	svc := scrape.Service{Browser: browser.PlaywrightBrowser{}}

	result, err := svc.Run(context.Background(), scrape.Request{
		Mode:       smbccard.ModeWebMeisaiHTMLDump,
		OutputPath: out,
		Headless:   true,
	})
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if result.Title == "" {
		t.Fatalf("Title = %q, want non-empty", result.Title)
	}

	data, err := os.ReadFile(out)
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}
	html := string(data)
	if !strings.Contains(html, "<html") {
		t.Fatalf("HTML = %q", html)
	}
	if !strings.Contains(html, "明細") {
		t.Fatalf("HTML did not contain expected statement marker: %q", html)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `nix develop -c bash -lc 'cd myscraper && go test ./e2e -run TestSMBCCardWebMeisaiSmoke -v'`
Expected: PASS with `SKIP` before implementation when env vars are absent, and PASS with `SKIP` after implementation in normal local development. When `PLAYWRIGHT_E2E_SMBCCARD=1` and valid credentials are supplied, it should execute the browser flow.

- [ ] **Step 3: Write minimal documentation**

Append this section to `README.md` under `## myscraper (Go)`:

```md
### myscraper SMBC Card WEB明細 HTML dump

Current scope: authenticated login and raw HTML snapshot only. CSV parsing for the WEB明細 table comes later.

Environment variables:

- `SMBC_VPASS_ID` and `SMBC_VPASS_PASSWORD`
- or the legacy fallback pair `user` and `pass`

Run:

```bash
nix develop
cd myscraper
go test ./...
go run ./cmd/myscraper --mode smbc-card-webmeisai --out tmp/smbccard-webmeisai.html
```

Optional real-browser smoke test:

```bash
nix develop
cd myscraper
PLAYWRIGHT_E2E_SMBCCARD=1 go test ./e2e -run TestSMBCCardWebMeisaiSmoke -v
```
```

- [ ] **Step 4: Run the full Go test suite**

Run: `nix develop -c bash -lc 'cd myscraper && go test ./...'`
Expected: PASS, with the SMBC smoke test reported as `SKIP` unless `PLAYWRIGHT_E2E_SMBCCARD=1` and credentials are set

- [ ] **Step 5: Commit**

```bash
git add README.md myscraper/e2e/smbccard_webmeisai_smoke_test.go
git commit -m "docs: add smbc card html dump usage"
```

## Self-Review

- Spec coverage: the plan covers CLI mode selection, env-based credential loading, authenticated navigation through `https://www.smbc-card.com/index.jsp` and `https://www.smbc-card.com/mem/index.jsp`, redirect verification for the mypage URL, navigation to `https://www.smbc-card.com/memx/web_meisai/top/index.html#info2`, and HTML snapshot output. CSV conversion is intentionally excluded from this increment and called out in the header and README.
- Placeholder scan: no `TODO`, `TBD`, or “handle appropriately” placeholders remain. The only deferred item is CSV parsing, which is explicitly outside scope rather than a hidden placeholder inside the tasks.
- Type consistency: `ModeWebMeisaiHTMLDump`, `Credentials`, `CaptureWebMeisai`, and `FetchSMBCCardWebMeisai` are named consistently across CLI parsing, service routing, browser implementation, and tests.

## Execution Handoff

Plan complete and saved to `docs/superpowers/plans/2026-06-04-myscraper-smbccard-webmeisai-html-dump.md`. Two execution options:

1. Subagent-Driven (recommended) - I dispatch a fresh subagent per task, review between tasks, fast iteration
2. Inline Execution - Execute tasks in this session using executing-plans, batch execution with checkpoints

Which approach?
