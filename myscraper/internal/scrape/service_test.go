package scrape

import (
	"context"
	"os"
	"path/filepath"
	"strings"
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

// TestServiceRunFetchModeWritesHTML verifies that fetch-url mode
// delegates to Browser.Fetch, writes the returned HTML to disk, and
// surfaces the snapshot's title on success.
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
		Mode:       ModeFetchURL,
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

// TestServiceRunSMBCModeWritesHTML verifies that smbc-card-webmeisai
// mode loads credentials from the environment, delegates to
// Browser.FetchSMBCCardWebMeisai with those credentials, and writes
// the returned HTML to disk.
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

// TestServiceRunRejectsUnknownMode verifies that Service.Run surfaces
// a clear "unsupported scrape mode" error for any mode value that
// neither fetch-url nor the SMBC flow handle, without touching the
// Browser or the filesystem.
func TestServiceRunRejectsUnknownMode(t *testing.T) {
	svc := Service{Browser: &fakeBrowser{}}

	_, err := svc.Run(context.Background(), Request{
		Mode:       "bogus-mode",
		OutputPath: filepath.Join(t.TempDir(), "x.html"),
		Headless:   true,
	})
	if err == nil {
		t.Fatalf("Run() error = nil, want unsupported-mode error")
	}
	if err.Error() != "unsupported scrape mode: bogus-mode" {
		t.Fatalf("Run() error = %q, want %q", err.Error(), "unsupported scrape mode: bogus-mode")
	}
}

// TestServiceRunSMBCModeReturnsMissingCredsError verifies that
// Service.Run fails fast with a "SMBC credentials are required" error
// when the operator has not provided any SMBC_VPASS_* or user/pass
// environment variables, without reaching the Browser.
func TestServiceRunSMBCModeReturnsMissingCredsError(t *testing.T) {
	t.Setenv("SMBC_VPASS_ID", "")
	t.Setenv("SMBC_VPASS_PASSWORD", "")
	t.Setenv("user", "")
	t.Setenv("pass", "")

	svc := Service{Browser: &fakeBrowser{}}

	_, err := svc.Run(context.Background(), Request{
		Mode:       smbccard.ModeWebMeisaiHTMLDump,
		OutputPath: filepath.Join(t.TempDir(), "x.html"),
		Headless:   true,
	})
	if err == nil {
		t.Fatalf("Run() error = nil, want missing-credentials error")
	}
	if !strings.Contains(err.Error(), "SMBC credentials are required") {
		t.Fatalf("Run() error = %q, want it to contain %q", err.Error(), "SMBC credentials are required")
	}
}
