package moneyforward

import (
	"os"
	"path/filepath"
	"testing"
)

func writeCookieFixture(t *testing.T, body string) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "cookie.json")
	if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
		t.Fatalf("write fixture: %v", err)
	}
	return path
}

func TestLoadCookiesNormalisesAndDropsInvalid(t *testing.T) {
	body := `[
		{
			"name": "sid", "value": "abc", "domain": ".moneyforward.com",
			"path": "/", "secure": true, "httpOnly": true,
			"expirationDate": 1700000000.5, "sameSite": "no_restriction"
		},
		{
			"name": "lo", "value": "x", "domain": ".moneyforward.com",
			"path": "/", "sameSite": "lax"
		},
		{
			"name": "weird", "value": "y", "domain": ".moneyforward.com",
			"path": "/", "sameSite": "bogus"
		},
		{"name": "missing-domain", "value": "z"}
	]`
	path := writeCookieFixture(t, body)
	got, err := LoadCookies(path)
	if err != nil {
		t.Fatalf("LoadCookies() error = %v", err)
	}
	if len(got) != 3 {
		t.Fatalf("len(cookies) = %d, want 3; got = %+v", len(got), got)
	}
	if got[0].Name != "sid" || got[0].SameSite != "None" || got[0].Expires != 1700000000.5 {
		t.Fatalf("cookie[0] = %+v", got[0])
	}
	if got[1].Name != "lo" || got[1].SameSite != "Lax" {
		t.Fatalf("cookie[1] = %+v", got[1])
	}
	if got[2].Name != "weird" || got[2].SameSite != "" {
		t.Fatalf("cookie[2] = %+v (want Name=weird, SameSite=\"\")", got[2])
	}
}

func TestLoadCookiesRejectsEmpty(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "cookie.json")
	if err := os.WriteFile(path, []byte("[]"), 0o600); err != nil {
		t.Fatalf("write fixture: %v", err)
	}
	if _, err := LoadCookies(path); err == nil {
		t.Fatalf("expected error for empty cookie file")
	}
}
