package nrkn

import (
	"context"
	"time"
)

type Session interface {
	Login(context.Context, string, string, string) error
	NavigateToAssets(context.Context) error
	BodyHTML(context.Context) (string, error)
	Logout(context.Context) error
	Close() error
}

type Credentials struct{ ID, Password, Birthday string }

type errText string

func (e errText) Error() string { return string(e) }

var ErrConcurrentLogin = errText("nrkn: concurrent login")

const loginURL = "https://www.nrkn.co.jp/webapp/nrk/"
const pageTimeout = 30 * time.Second
