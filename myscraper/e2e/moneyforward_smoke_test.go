package e2e

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/azuki774/myscrapers/myscraper/internal/moneyforward"
)

// TestMoneyforwardSmoke drives moneyforward.Fetch end-to-end through
// the table extractors, CSV writers, and UTF-8→Shift-JIS round-trip
// against checked-in HTML fixtures. Gated on MF_E2E=1 so the default
// `go test ./...` stays offline.
func TestMoneyforwardSmoke(t *testing.T) {
	if os.Getenv("MF_E2E") != "1" {
		t.Skip("set MF_E2E=1 to run moneyforward smoke test")
	}

	cfHTML, err := os.ReadFile(filepath.Join("testdata", "moneyforward_cf.html"))
	if err != nil {
		t.Fatalf("read cf fixture: %v", err)
	}
	bsHTML, err := os.ReadFile(filepath.Join("testdata", "moneyforward_bs.html"))
	if err != nil {
		t.Fatalf("read bs fixture: %v", err)
	}

	dir := t.TempDir()
	cookiePath := filepath.Join(dir, "cookie.json")
	if err := os.WriteFile(cookiePath, []byte(`[{"name":"sid","value":"abc","domain":".moneyforward.com","path":"/","sameSite":"lax"}]`), 0o600); err != nil {
		t.Fatalf("write cookie: %v", err)
	}
	cookies, err := moneyforward.LoadCookies(cookiePath)
	if err != nil {
		t.Fatalf("LoadCookies() error = %v", err)
	}

	sess := moneyforward.NewFakeSession([]string{string(cfHTML), string(cfHTML), string(bsHTML)})

	if err := moneyforward.Fetch(context.Background(), moneyforward.FetchOptions{
		Session:   sess,
		Cookies:   cookies,
		OutputDir: dir,
		Now:       time.Date(2024, 12, 15, 0, 0, 0, 0, time.UTC),
	}); err != nil {
		t.Fatalf("Fetch() error = %v", err)
	}

	cf, err := os.ReadFile(filepath.Join(dir, "cf.csv"))
	if err != nil {
		t.Fatalf("read cf.csv: %v", err)
	}
	if !strings.Contains(string(cf), "2024/11/28") {
		t.Fatalf("cf.csv missing expected date, got %q", string(cf))
	}

	last, err := os.ReadFile(filepath.Join(dir, "cf_lastmonth.csv"))
	if err != nil {
		t.Fatalf("read cf_lastmonth.csv: %v", err)
	}
	if !strings.Contains(string(last), "2024/11/28") {
		t.Fatalf("cf_lastmonth.csv missing expected date, got %q", string(last))
	}

	asset, err := os.ReadFile(filepath.Join(dir, "asset_history.csv"))
	if err != nil {
		t.Fatalf("read asset_history.csv: %v", err)
	}
	if !strings.Contains(string(asset), "1,234,567") {
		t.Fatalf("asset_history.csv missing amount, got %q", string(asset))
	}
}
