package moneyforward

import (
	"context"
	"errors"
	"time"
)

// fakeSession records every call in order and replays scripted responses
// for Content(). It is the shared test double for both in-package tests
// (which construct it directly to inspect recorded calls) and external
// test packages that use the NewFakeSession constructor; it lives in a
// non-test file so external packages such as e2e can reach it through
// the exported constructor without re-implementing the Session
// interface.
type fakeSession struct {
	calls      []string
	gotCookies []Cookie
	clicks     []string
	waited     []time.Duration
	contentSeq []string
	contentIdx int
	closed     bool
	contentErr error
	clickErr   error
}

// Compile-time check that fakeSession satisfies Session.
var _ Session = (*fakeSession)(nil)

// NewFakeSession returns a Session that yields the given strings from
// successive Content() calls, and records every other call without
// failing. It is the constructor external test packages should use to
// obtain a scripted Session double; in-package tests construct
// fakeSession directly so they can inspect the recorded calls/clicks/
// waited slices.
func NewFakeSession(contentSeq []string) Session {
	return &fakeSession{contentSeq: contentSeq}
}

func (f *fakeSession) AddCookies(_ context.Context, cs []Cookie) error {
	f.calls = append(f.calls, "AddCookies")
	f.gotCookies = append(f.gotCookies, cs...)
	return nil
}

func (f *fakeSession) Goto(_ context.Context, url string) error {
	f.calls = append(f.calls, "Goto:"+url)
	return nil
}

func (f *fakeSession) Content(_ context.Context) (string, error) {
	f.calls = append(f.calls, "Content")
	if f.contentErr != nil {
		return "", f.contentErr
	}
	if f.contentIdx >= len(f.contentSeq) {
		return "", errors.New("fakeSession: no more scripted content")
	}
	out := f.contentSeq[f.contentIdx]
	f.contentIdx++
	return out, nil
}

func (f *fakeSession) ClickByXPath(_ context.Context, xp string) error {
	f.calls = append(f.calls, "ClickByXPath:"+xp)
	f.clicks = append(f.clicks, "xpath:"+xp)
	return f.clickErr
}

func (f *fakeSession) ClickByText(_ context.Context, text string) error {
	f.calls = append(f.calls, "ClickByText:"+text)
	f.clicks = append(f.clicks, "text:"+text)
	return f.clickErr
}

func (f *fakeSession) ClickLinkIn(_ context.Context, parent, link string) error {
	f.calls = append(f.calls, "ClickLinkIn:"+parent+":"+link)
	f.clicks = append(f.clicks, "linkin:"+parent+":"+link)
	return f.clickErr
}

func (f *fakeSession) Wait(_ context.Context, d time.Duration) error {
	f.calls = append(f.calls, "Wait")
	f.waited = append(f.waited, d)
	return nil
}

func (f *fakeSession) Close() error {
	f.calls = append(f.calls, "Close")
	f.closed = true
	return nil
}
