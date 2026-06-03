# Myscraper Go Playwright Bootstrap Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build a standalone Go-based scraper bootstrap under `myscraper` with a `myscraper` binary, Playwright-backed page fetching, and a Nix development shell, while leaving the existing Python scrapers untouched.

**Architecture:** Keep the new Go code isolated inside `myscraper` as its own module so the current Python layout can remain unchanged. Split the Go code into a thin CLI parser, a service layer that writes scrape output, and a Playwright-backed browser adapter behind an interface so unit tests do not need a real browser. Put Nix at the repository root so `nix develop` can provision Go, Node, and the Playwright browser install path for the whole checkout without forcing a rewrite of the current build setup.

**Tech Stack:** Go 1.24, `github.com/playwright-community/playwright-go` v0.5200.1, Nix flakes, direnv, Go standard library, `net/http/httptest`

---

## Planned File Structure

- Create: `flake.nix`
- Create: `.envrc`
- Modify: `.gitignore`
- Modify: `README.md`
- Create: `myscraper/go.mod`
- Create: `myscraper/cmd/myscraper/main.go`
- Create: `myscraper/internal/cli/fetch.go`
- Create: `myscraper/internal/cli/fetch_test.go`
- Create: `myscraper/internal/cli/run.go`
- Create: `myscraper/internal/cli/run_test.go`
- Create: `myscraper/internal/scrape/service.go`
- Create: `myscraper/internal/scrape/service_test.go`
- Create: `myscraper/internal/browser/playwright.go`
- Create: `myscraper/e2e/fetch_smoke_test.go`
- Create: `myscraper/scripts/install-playwright.sh`
- Create: `myscraper/scripts/dev-shell-smoke.sh`

`myscraper` is intentionally separate from the existing `src/` tree because `src/` is Python today. That keeps the Go module, Playwright cache, and future Go-only tests from colliding with the current Python import layout.

### Task 1: Bootstrap the Go Module and Fetch Flag Parsing

**Files:**
- Create: `myscraper/go.mod`
- Create: `myscraper/cmd/myscraper/main.go`
- Create: `myscraper/internal/cli/fetch.go`
- Test: `myscraper/internal/cli/fetch_test.go`

- [ ] **Step 1: Write the failing test**

```go
package cli

import "testing"

func TestParseArgs(t *testing.T) {
	t.Run("requires url", func(t *testing.T) {
		_, err := ParseArgs([]string{"--out", "tmp/custom.html"})
		if err == nil || err.Error() != "--url is required" {
			t.Fatalf("expected url validation error, got %v", err)
		}
	})

	t.Run("applies defaults", func(t *testing.T) {
		got, err := ParseArgs([]string{"--url", "https://example.com"})
		if err != nil {
			t.Fatalf("ParseArgs() error = %v", err)
		}
		if got.URL != "https://example.com" {
			t.Fatalf("URL = %q, want %q", got.URL, "https://example.com")
		}
		if got.OutputPath != "tmp/example.html" {
			t.Fatalf("OutputPath = %q, want %q", got.OutputPath, "tmp/example.html")
		}
		if !got.Headless {
			t.Fatalf("Headless = %v, want true", got.Headless)
		}
	})
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd myscraper && go test ./internal/cli -run TestParseArgs -v`
Expected: FAIL with `undefined: ParseArgs`

- [ ] **Step 3: Write minimal implementation**

`myscraper/go.mod`

```go
module github.com/azuki774/myscrapers/myscraper

go 1.24.0
```

`myscraper/internal/cli/fetch.go`

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
	fs.StringVar(&opts.OutputPath, "out", "tmp/example.html", "path to write HTML snapshot")
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

`myscraper/cmd/myscraper/main.go`

```go
package main

import (
	"fmt"
	"os"

	"github.com/azuki774/myscrapers/myscraper/internal/cli"
)

func main() {
	opts, err := cli.ParseArgs(os.Args[1:])
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(2)
	}

	fmt.Fprintf(os.Stdout, "myscraper scaffold ready: %s -> %s\n", opts.URL, opts.OutputPath)
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `cd myscraper && go test ./internal/cli -run TestParseArgs -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add myscraper/go.mod myscraper/cmd/myscraper/main.go myscraper/internal/cli/fetch.go myscraper/internal/cli/fetch_test.go
git commit -m "feat: scaffold myscraper Go module"
```

### Task 2: Add the Scrape Service and CLI Execution Flow

**Files:**
- Create: `myscraper/internal/cli/run.go`
- Test: `myscraper/internal/cli/run_test.go`
- Create: `myscraper/internal/scrape/service.go`
- Test: `myscraper/internal/scrape/service_test.go`
- Modify: `myscraper/cmd/myscraper/main.go`

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
				URL:   "https://example.com",
				Title: "Example Domain",
				HTML:  "<html><body><h1>Example Domain</h1></body></html>",
			},
		},
	}

	result, err := svc.Run(context.Background(), Request{
		URL:        "https://example.com",
		OutputPath: out,
		Headless:   true,
	})
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if result.Title != "Example Domain" {
		t.Fatalf("Title = %q, want %q", result.Title, "Example Domain")
	}

	data, err := os.ReadFile(out)
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}
	if string(data) != "<html><body><h1>Example Domain</h1></body></html>" {
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
		[]string{"--url", "https://example.com", "--out", "tmp/page.html"},
		stdout,
		stderr,
		fakeRunner{
			result: scrape.Result{
				Title:      "Example Domain",
				OutputPath: "tmp/page.html",
			},
		},
	)

	if exitCode != 0 {
		t.Fatalf("exitCode = %d, want 0; stderr = %q", exitCode, stderr.String())
	}
	if stdout.String() != "saved tmp/page.html (Example Domain)\n" {
		t.Fatalf("stdout = %q", stdout.String())
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `cd myscraper && go test ./internal/... -run 'Test(ServiceRunWritesHTML|RunPrintsSavedPath)' -v`
Expected: FAIL with `undefined: Service` and `undefined: Run`

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

Run: `cd myscraper && go test ./internal/... -run 'Test(ServiceRunWritesHTML|RunPrintsSavedPath)' -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add myscraper/internal/cli/run.go myscraper/internal/cli/run_test.go myscraper/internal/scrape/service.go myscraper/internal/scrape/service_test.go myscraper/cmd/myscraper/main.go
git commit -m "feat: add myscraper service flow"
```

### Task 3: Wire the Real Playwright Browser Adapter

**Files:**
- Create: `myscraper/internal/browser/playwright.go`
- Modify: `myscraper/cmd/myscraper/main.go`
- Test: `myscraper/e2e/fetch_smoke_test.go`

- [ ] **Step 1: Write the failing smoke test**

```go
package e2e

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/azuki774/myscrapers/myscraper/internal/browser"
	"github.com/azuki774/myscrapers/myscraper/internal/scrape"
)

func TestFetchSmoke(t *testing.T) {
	if os.Getenv("PLAYWRIGHT_E2E") != "1" {
		t.Skip("set PLAYWRIGHT_E2E=1 to run browser smoke test")
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = w.Write([]byte("<html><head><title>smoke</title></head><body><h1>hello</h1></body></html>"))
	}))
	defer server.Close()

	out := filepath.Join(t.TempDir(), "page.html")
	svc := scrape.Service{Browser: browser.PlaywrightBrowser{}}

	result, err := svc.Run(context.Background(), scrape.Request{
		URL:        server.URL,
		OutputPath: out,
		Headless:   true,
	})
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if result.Title != "smoke" {
		t.Fatalf("Title = %q, want %q", result.Title, "smoke")
	}

	data, err := os.ReadFile(out)
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}
	if !strings.Contains(string(data), "<h1>hello</h1>") {
		t.Fatalf("HTML = %q", string(data))
	}
}
```

- [ ] **Step 2: Run smoke test to verify it fails**

Run: `cd myscraper && PLAYWRIGHT_E2E=1 go test ./e2e -run TestFetchSmoke -v`
Expected: FAIL with `cannot find package` / `undefined: browser.PlaywrightBrowser`

- [ ] **Step 3: Write minimal implementation**

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

- [ ] **Step 4: Run tests to verify they pass**

Run: `cd myscraper && go test ./internal/... -v`
Expected: PASS

Run: `cd myscraper && go run github.com/playwright-community/playwright-go/cmd/playwright@v0.5200.1 install chromium && PLAYWRIGHT_E2E=1 go test ./e2e -run TestFetchSmoke -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add myscraper/internal/browser/playwright.go myscraper/e2e/fetch_smoke_test.go myscraper/cmd/myscraper/main.go
git commit -m "feat: wire playwright browser adapter"
```

### Task 4: Add the Nix Shell, Playwright Installer, and Repo Documentation

**Files:**
- Create: `flake.nix`
- Create: `.envrc`
- Modify: `.gitignore`
- Modify: `README.md`
- Create: `myscraper/scripts/install-playwright.sh`
- Create: `myscraper/scripts/dev-shell-smoke.sh`

- [ ] **Step 1: Write the failing shell smoke check**

`myscraper/scripts/dev-shell-smoke.sh`

```bash
#!/usr/bin/env bash
set -euo pipefail

nix develop -c bash -lc '
  test -n "${PLAYWRIGHT_BROWSERS_PATH:-}"
  command -v go >/dev/null
  command -v node >/dev/null
'
```

- [ ] **Step 2: Run smoke check to verify it fails**

Run: `bash myscraper/scripts/dev-shell-smoke.sh`
Expected: FAIL because `flake.nix` does not exist yet

- [ ] **Step 3: Write minimal implementation**

`flake.nix`

```nix
{
  description = "Development shell for myscrapers";

  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/nixos-26.05";
    flake-utils.url = "github:numtide/flake-utils";
  };

  outputs = { self, nixpkgs, flake-utils }:
    flake-utils.lib.eachDefaultSystem (system:
      let
        pkgs = import nixpkgs { inherit system; };
      in {
        devShells.default = pkgs.mkShell {
          packages = with pkgs; [
            bashInteractive
            coreutils
            direnv
            go_1_24
            golangci-lint
            gopls
            gotestsum
            nodejs_24
            nixfmt-rfc-style
          ];

          shellHook = ''
            export GOPATH="$PWD/.gopath"
            export GOMODCACHE="$PWD/.gopath/pkg/mod"
            export PLAYWRIGHT_BROWSERS_PATH="$PWD/myscraper/.playwright"
            export PATH="$PWD/myscraper/bin:$PATH"
          '';
        };
      });
}
```

`.envrc`

```bash
use flake
```

`.gitignore`

```gitignore
.direnv/
.gopath/
myscraper/.playwright/
myscraper/tmp/
```

`myscraper/scripts/install-playwright.sh`

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
## myscraper (Go + Playwright)

The Go scraper lives in `myscraper` and is separate from the existing Python scrapers.

### Development setup

```bash
nix develop
direnv allow
cd myscraper
go test ./internal/... -v
./scripts/install-playwright.sh
PLAYWRIGHT_E2E=1 go test ./e2e -run TestFetchSmoke -v
go run ./cmd/myscraper --url https://example.com --out tmp/example.html
```
````

- [ ] **Step 4: Run checks to verify they pass**

Run: `bash myscraper/scripts/dev-shell-smoke.sh`
Expected: PASS

Run: `nix develop -c bash -lc 'cd myscraper && go test ./internal/... -v'`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add flake.nix .envrc .gitignore README.md myscraper/scripts/install-playwright.sh myscraper/scripts/dev-shell-smoke.sh
git commit -m "chore: add nix shell for myscraper"
```

## Self-Review

- Spec coverage: the plan creates a dedicated Go directory, uses Playwright for the scraping path, introduces Nix for the development workflow, keeps the binary name `myscraper`, and does not modify the existing Python scraper implementation.
- Placeholder scan: no `TBD`, `TODO`, or cross-references that require reading another task to complete a step.
- Type consistency: `cli.Options`, `scrape.Request`, `scrape.Result`, `scrape.PageSnapshot`, and `browser.PlaywrightBrowser` keep the same names across every task.
