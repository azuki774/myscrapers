package sbi

import (
	"context"
	"time"
)

// Session is the minimum surface the SBI orchestrator needs from a
// browser: restore the saved passkey into a virtual authenticator, log
// in with it, navigate to fixed pages, and read page text.
type Session interface {
	// LoginWithPasskey restores the given credentials into a WebAuthn
	// virtual authenticator, navigates to the SBI login URL, and waits
	// until the authenticated portal (ETGate) loads. It returns
	// ErrLoginFailed when the passkey was not accepted.
	LoginWithPasskey(ctx context.Context, passkey *PasskeyFile) error
	Goto(ctx context.Context, url string) error
	// BodyText returns the trimmed innerText of the document body.
	BodyText(ctx context.Context) (string, error)
	Wait(ctx context.Context, d time.Duration) error
	Close() error
}
