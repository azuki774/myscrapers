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
