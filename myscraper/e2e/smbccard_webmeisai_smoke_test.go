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

// TestSMBCCardWebMeisaiSmoke is an end-to-end check that the SMBC Card
// authenticated flow can log in to Vpass, reach the WEB明細 page, and
// write a non-empty HTML snapshot that mentions the expected
// statement marker. The test is gated on PLAYWRIGHT_E2E_SMBCCARD=1
// and the presence of Vpass credentials so unit-only runs stay
// network-free and credential-free.
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
