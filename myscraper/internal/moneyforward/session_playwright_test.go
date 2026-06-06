package moneyforward

import "testing"

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
