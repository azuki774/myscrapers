# Myscraper Minimum CLI And GitHub Smoke Test Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build the minimum useful `myscraper` CLI and add a Playwright-backed smoke test that opens `https://github.com`.

**Architecture:** Keep the CLI thin: argument parsing and exit codes stay in `internal/cli`, scrape orchestration lives in `internal/scrape`, and the real browser integration lives in `internal/browser`. Cover the browser-free path with unit tests and gate the real GitHub network/browser test behind `PLAYWRIGHT_E2E=1` so normal `go test` stays fast and deterministic.

**Tech Stack:** Go 1.24, `github.com/playwright-community/playwright-go` v0.5200.1, Nix dev shell, Go standard library

---

## Planned File Structure

- Modify: `myscraper/go.mod`
- Create: `myscraper/go.sum`
- Modify: `myscraper/cmd/myscraper/main.go`
- Create: `myscraper/internal/cli/options.go`
- Create: `myscraper/internal/cli/options_test.go`
- Modify: `myscraper/internal/cli/run.go`
- Modify: `myscraper/internal/cli/run_test.go`
- Create: `myscraper/internal/scrape/service.go`
- Create: `myscraper/internal/scrape/service_test.go`
- Create: `myscraper/internal/browser/playwright.go`
- Create: `myscraper/e2e/github_smoke_test.go`
- Modify: `README.md`

### Task 1: Add CLI Argument Parsing

**Files:**
- Create: `myscraper/internal/cli/options.go`
- Create: `myscraper/internal/cli/options_test.go`

- [ ] **Step 1: Write the failing test**

`myscraper/internal/cli/options_test.go`

```go
package cli

import "testing"

func TestParseArgs(t *testing.T) {
	t.Run("requires url", func(t *testing.T) {
		_, err := ParseArgs([]string{"--out", "tmp/page.html"})
		if err == nil || err.Error() != "--url is required" {
			t.Fatalf("expected url validation error, got %v", err)
		}
	})

	t.Run("applies defaults", func(t *testing.T) {
		got, err := ParseArgs([]string{"--url", "https://github.com"})
		if err != nil {
			t.Fatalf("ParseArgs() error = %v", err)
		}
		if got.URL != "https://github.com" {
			t.Fatalf("URL = %q, want %q", got.URL, "https://github.com")
		}
		if got.OutputPath != "tmp/page.html" {
			t.Fatalf("OutputPath = %q, want %q", got.OutputPath, "tmp/page.html")
		}
		if !got.Headless {
			t.Fatalf("Headless = %v, want true", got.Headless)
		}
	})
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd myscraper && nix develop -c go test ./internal/cli -run TestParseArgs -v`
Expected: FAIL with `undefined: ParseArgs`

- [ ] **Step 3: Write minimal implementation**

`myscraper/internal/cli/options.go`

```go
package cli

import (
	"errors"
	"flag"
	"io"
)

type Options struct {
	URL        string
	OutputPath string
	Headless   bool
}

func ParseArgs(args []string) (Options, error) {
	fs := flag.NewFlagSet("myscraper", flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	opts := Options{}
	fs.StringVar(&opts.URL, "url", "", "target URL to scrape")
	fs.StringVar(&opts.OutputPath, "out", "tmp/page.html", "path to write HTML snapshot")
	fs.BoolVar(&opts.Headless, "headless", true, "launch browser in headless mode")

	if err := fs.Parse(args); err != nil {
		return Options{}, err
	}
	if opts.URL == "" {
		return Options{}, errors.New("--url is required")
	}
	return opts, nil
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `cd myscraper && nix develop -c go test ./internal/cli -run TestParseArgs -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add myscraper/internal/cli/options.go myscraper/internal/cli/options_test.go
git commit -m "feat: add myscraper cli argument parsing"
```

### Task 2: Add the Scrape Service and CLI Execution Flow

**Files:**
- Modify: `myscraper/cmd/myscraper/main.go`
- Modify: `myscraper/internal/cli/run.go`
- Modify: `myscraper/internal/cli/run_test.go`
- Create: `myscraper/internal/scrape/service.go`
- Create: `myscraper/internal/scrape/service_test.go`

- [ ] **Step 1: Write the failing tests**

`myscraper/internal/scrape/service_test.go`

```go
package scrape

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

type fakeBrowser struct {
	snapshot PageSnapshot
	err      error
}

func (f fakeBrowser) Fetch(ctx context.Context, url string, headless bool) (PageSnapshot, error) {
	return f.snapshot, f.err
}

func TestServiceRunWritesHTML(t *testing.T) {
	out := filepath.Join(t.TempDir(), "page.html")
	svc := Service{
		Browser: fakeBrowser{
			snapshot: PageSnapshot{
				URL:   "https://github.com",
				Title: "GitHub",
				HTML:  "<html><body><h1>GitHub</h1></body></html>",
			},
		},
	}

	result, err := svc.Run(context.Background(), Request{
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

	data, err := os.ReadFile(out)
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}
	if string(data) != "<html><body><h1>GitHub</h1></body></html>" {
		t.Fatalf("HTML = %q", string(data))
	}
}
```

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
}

func (f fakeRunner) Run(ctx context.Context, req scrape.Request) (scrape.Result, error) {
	return f.result, f.err
}

func TestRunPrintsSavedPath(t *testing.T) {
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}

	exitCode := Run(
		[]string{"--url", "https://github.com", "--out", "tmp/github.html"},
		stdout,
		stderr,
		fakeRunner{
			result: scrape.Result{
				Title:      "GitHub",
				OutputPath: "tmp/github.html",
			},
		},
	)

	if exitCode != 0 {
		t.Fatalf("exitCode = %d, want 0; stderr = %q", exitCode, stderr.String())
	}
	if stdout.String() != "saved tmp/github.html (GitHub)\n" {
		t.Fatalf("stdout = %q", stdout.String())
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `cd myscraper && nix develop -c go test ./internal/... -run 'Test(ServiceRunWritesHTML|RunPrintsSavedPath)' -v`
Expected: FAIL with `undefined: Service` and `too many arguments in call to Run`

- [ ] **Step 3: Write minimal implementation**

`myscraper/internal/scrape/service.go`

```go
package scrape

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
)

type Browser interface {
	Fetch(ctx context.Context, url string, headless bool) (PageSnapshot, error)
}

type PageSnapshot struct {
	URL   string
	Title string
	HTML  string
}

type Request struct {
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

	snapshot, err := s.Browser.Fetch(ctx, req.URL, req.Headless)
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
		URL:        opts.URL,
		OutputPath: opts.OutputPath,
		Headless:   opts.Headless,
	})
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}

	fmt.Fprintf(stdout, "saved %s (%s)\n", result.OutputPath, result.Title)
	return 0
}
```

`myscraper/cmd/myscraper/main.go`

```go
package main

import (
	"context"
	"fmt"
	"os"

	"github.com/azuki774/myscrapers/myscraper/internal/cli"
	"github.com/azuki774/myscrapers/myscraper/internal/scrape"
)

type notReadyRunner struct{}

func (notReadyRunner) Run(_ context.Context, _ scrape.Request) (scrape.Result, error) {
	return scrape.Result{}, fmt.Errorf("browser wiring is not ready yet")
}

func main() {
	os.Exit(cli.Run(os.Args[1:], os.Stdout, os.Stderr, notReadyRunner{}))
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `cd myscraper && nix develop -c go test ./internal/... -run 'Test(ServiceRunWritesHTML|RunPrintsSavedPath)' -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add myscraper/cmd/myscraper/main.go myscraper/internal/cli/run.go myscraper/internal/cli/run_test.go myscraper/internal/scrape/service.go myscraper/internal/scrape/service_test.go
git commit -m "feat: add myscraper scrape service flow"
```

### Task 3: Wire Playwright And Add The GitHub Smoke Test

**Files:**
- Modify: `myscraper/go.mod`
- Create: `myscraper/go.sum`
- Modify: `myscraper/cmd/myscraper/main.go`
- Create: `myscraper/internal/browser/playwright.go`
- Create: `myscraper/e2e/github_smoke_test.go`
- Modify: `README.md`

- [ ] **Step 1: Write the failing smoke test**

`myscraper/e2e/github_smoke_test.go`

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
)

func TestGitHubSmoke(t *testing.T) {
	if os.Getenv("PLAYWRIGHT_E2E") != "1" {
		t.Skip("set PLAYWRIGHT_E2E=1 to run browser smoke test")
	}

	out := filepath.Join(t.TempDir(), "github.html")
	svc := scrape.Service{Browser: browser.PlaywrightBrowser{}}

	result, err := svc.Run(context.Background(), scrape.Request{
		URL:        "https://github.com",
		OutputPath: out,
		Headless:   true,
	})
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if !strings.Contains(result.Title, "GitHub") {
		t.Fatalf("Title = %q, want to contain %q", result.Title, "GitHub")
	}

	data, err := os.ReadFile(out)
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}
	html := string(data)
	if !strings.Contains(html, "<html") {
		t.Fatalf("HTML = %q", html)
	}
	if !strings.Contains(html, "github") && !strings.Contains(html, "GitHub") {
		t.Fatalf("HTML did not contain expected github marker: %q", html)
	}
}
```

- [ ] **Step 2: Run smoke test to verify it fails**

Run: `cd myscraper && PLAYWRIGHT_E2E=1 nix develop -c go test ./e2e -run TestGitHubSmoke -v`
Expected: FAIL with `cannot find module providing package github.com/playwright-community/playwright-go` or `undefined: browser.PlaywrightBrowser`

- [ ] **Step 3: Write minimal implementation**

`myscraper/go.mod`

```go
module github.com/azuki774/myscrapers/myscraper

go 1.24.0

require github.com/playwright-community/playwright-go v0.5200.1
```

`myscraper/internal/browser/playwright.go`

```go
package browser

import (
	"context"
	"fmt"

	"github.com/azuki774/myscrapers/myscraper/internal/scrape"
	"github.com/playwright-community/playwright-go"
)

type PlaywrightBrowser struct{}

func (PlaywrightBrowser) Fetch(ctx context.Context, url string, headless bool) (scrape.PageSnapshot, error) {
	pw, err := playwright.Run()
	if err != nil {
		return scrape.PageSnapshot{}, fmt.Errorf("start playwright: %w", err)
	}
	defer pw.Stop()

	browser, err := pw.Chromium.Launch(playwright.BrowserTypeLaunchOptions{
		Headless: playwright.Bool(headless),
	})
	if err != nil {
		return scrape.PageSnapshot{}, fmt.Errorf("launch chromium: %w", err)
	}
	defer browser.Close()

	page, err := browser.NewPage()
	if err != nil {
		return scrape.PageSnapshot{}, fmt.Errorf("new page: %w", err)
	}

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
		URL:   url,
		Title: title,
		HTML:  html,
	}, nil
}
```

`myscraper/cmd/myscraper/main.go`

```go
package main

import (
	"os"

	"github.com/azuki774/myscrapers/myscraper/internal/browser"
	"github.com/azuki774/myscrapers/myscraper/internal/cli"
	"github.com/azuki774/myscrapers/myscraper/internal/scrape"
)

func main() {
	os.Exit(cli.Run(
		os.Args[1:],
		os.Stdout,
		os.Stderr,
		scrape.Service{Browser: browser.PlaywrightBrowser{}},
	))
}
```


```bash
#!/usr/bin/env bash
set -euo pipefail

cd "$(dirname "$0")/.."

PLAYWRIGHT_VERSION="$(go list -m -f '{{.Version}}' github.com/playwright-community/playwright-go)"
if [[ -z "${PLAYWRIGHT_VERSION}" ]]; then
  echo "unable to determine playwright-go version" >&2
  exit 1
fi

export PLAYWRIGHT_BROWSERS_PATH="${PLAYWRIGHT_BROWSERS_PATH:-$PWD/.playwright}"
go run "github.com/playwright-community/playwright-go/cmd/playwright@${PLAYWRIGHT_VERSION}" install chromium
```

`README.md` addition

````markdown
### myscraper CLI

```bash
nix develop
cd myscraper
go test ./internal/... -v
PLAYWRIGHT_E2E=1 go test ./e2e -run TestGitHubSmoke -v
go run ./cmd/myscraper --url https://github.com --out tmp/github.html
```
````

- [ ] **Step 4: Run tests to verify they pass**

Run: `cd myscraper && nix develop -c go test ./internal/... -v`
Expected: PASS

Run: `cd myscraper && nix develop -c go test ./cmd/myscraper -v`
Expected: PASS or `[no test files]` with exit 0

Run: `cd myscraper && nix develop -c go run github.com/playwright-community/playwright-go/cmd/playwright@v0.5200.1 install chromium`
Expected: PASS

Run: `cd myscraper && PLAYWRIGHT_E2E=1 nix develop -c go test ./e2e -run TestGitHubSmoke -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add myscraper/go.mod myscraper/go.sum myscraper/cmd/myscraper/main.go myscraper/internal/browser/playwright.go myscraper/e2e/github_smoke_test.go README.md
git commit -m "feat: add playwright github smoke test"
```

## Self-Review

- Spec coverage: the plan adds a real CLI, writes fetched HTML to disk, uses Playwright to open `https://github.com`, and keeps the real browser/network test opt-in.
- Placeholder scan: there are no `TBD`, `TODO`, or “similar to” references that require guessing.
- Type consistency: `cli.Options`, `scrape.Request`, `scrape.Result`, `scrape.PageSnapshot`, and `browser.PlaywrightBrowser` are named consistently across all tasks.
