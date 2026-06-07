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
