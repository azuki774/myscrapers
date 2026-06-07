package moneyforward

import (
	"context"
	"errors"
	"testing"

	"github.com/playwright-community/playwright-go"
)

// TestPlaywrightSessionCloseIsIdempotent exercises the nil-out logic in
// Close() against a zero-value PlaywrightSession: the first call must
// succeed because all underlying fields are already nil, and the
// second call must also return nil because Close nil-ed any fields it
// successfully closed. This guards against a regression where a
// defer-Close + early-error Close would observe a second-time error
// from the underlying Playwright objects.
func TestPlaywrightSessionCloseIsIdempotent(t *testing.T) {
	s := &PlaywrightSession{}
	if err := s.Close(); err != nil {
		t.Fatalf("first Close() error = %v, want nil (all fields are zero-value)", err)
	}
	if err := s.Close(); err != nil {
		t.Fatalf("second Close() error = %v, want nil (fields were nil-ed by first Close)", err)
	}
}

func TestNewPlaywrightSessionInstallsDriverBeforeRun(t *testing.T) {
	t.Cleanup(func() {
		installPlaywrightDriver = defaultInstallPlaywrightDriver
		runPlaywright = defaultRunPlaywright
	})

	installErr := errors.New("boom")
	runCalled := false
	installPlaywrightDriver = func() error {
		return installErr
	}
	runPlaywright = func(_ ...*playwright.RunOptions) (*playwright.Playwright, error) {
		runCalled = true
		return nil, nil
	}

	_, err := NewPlaywrightSession(context.Background(), true)
	if !errors.Is(err, installErr) {
		t.Fatalf("NewPlaywrightSession() error = %v, want wrapped %v", err, installErr)
	}
	if runCalled {
		t.Fatal("NewPlaywrightSession() called playwright.Run() despite driver install failure")
	}
}

func TestPlaywrightLaunchOptionsMatchLegacySeleniumSetup(t *testing.T) {
	opts := launchOptions("/usr/bin/google-chrome", true)
	if opts.ExecutablePath == nil || *opts.ExecutablePath != "/usr/bin/google-chrome" {
		t.Fatalf("ExecutablePath = %v, want /usr/bin/google-chrome", opts.ExecutablePath)
	}
	if opts.Headless == nil || !*opts.Headless {
		t.Fatalf("Headless = %v, want true", opts.Headless)
	}
	wantArgs := []string{"--no-sandbox", "--disable-gpu", "--lang=ja-JP", "--disable-dev-shm-usage"}
	for _, want := range wantArgs {
		found := false
		for _, got := range opts.Args {
			if got == want {
				found = true
				break
			}
		}
		if !found {
			t.Fatalf("launch args = %v, want to contain %q", opts.Args, want)
		}
	}
}

func TestPlaywrightContextOptionsUseLegacyUserAgent(t *testing.T) {
	opts := contextOptions()
	if opts.UserAgent == nil {
		t.Fatal("UserAgent = nil, want MoneyForward-compatible UA")
	}
	if *opts.UserAgent != moneyforwardUserAgent {
		t.Fatalf("UserAgent = %q, want %q", *opts.UserAgent, moneyforwardUserAgent)
	}
	if opts.Locale == nil || *opts.Locale != "ja-JP" {
		t.Fatalf("Locale = %v, want ja-JP", opts.Locale)
	}
}
