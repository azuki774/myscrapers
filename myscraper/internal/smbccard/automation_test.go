package smbccard

import (
	"errors"
	"reflect"
	"testing"
)

type fakePage struct {
	steps   []string
	title   string
	html    string
	url     string
	waitErr error
}

func (p *fakePage) Goto(url string) error {
	p.steps = append(p.steps, "goto:"+url)
	p.url = url
	return nil
}

func (p *fakePage) FillByLabel(label, value string) error {
	p.steps = append(p.steps, "fill:"+label+"="+value)
	return nil
}

func (p *fakePage) ClickButton(name string) error {
	p.steps = append(p.steps, "click:"+name)
	return nil
}

func (p *fakePage) WaitForURL(pattern string) error {
	p.steps = append(p.steps, "wait:"+pattern)
	if p.waitErr != nil {
		return p.waitErr
	}
	if pattern == MyPageURLPattern {
		p.url = MyPageURL
	}
	if pattern == WebMeisaiURLPattern {
		p.url = WebMeisaiURL
	}
	return nil
}

func (p *fakePage) Title() (string, error) {
	return p.title, nil
}

func (p *fakePage) Content() (string, error) {
	return p.html, nil
}

func (p *fakePage) URL() string {
	return p.url
}

func TestCaptureWebMeisai(t *testing.T) {
	page := &fakePage{
		title: "WEB明細",
		html:  "<html><body>statement</body></html>",
	}

	got, err := CaptureWebMeisai(page, Credentials{
		LoginID:  "member-id",
		Password: "member-pass",
	})
	if err != nil {
		t.Fatalf("CaptureWebMeisai() error = %v", err)
	}

	wantSteps := []string{
		"goto:" + TopURL,
		"goto:" + LoginURL,
		"fill:VpassID=member-id",
		"fill:パスワード=member-pass",
		"click:ログイン",
		"wait:" + MyPageURLPattern,
		"goto:" + WebMeisaiURL,
		"wait:" + WebMeisaiURLPattern,
	}
	if !reflect.DeepEqual(page.steps, wantSteps) {
		t.Fatalf("steps = %#v, want %#v", page.steps, wantSteps)
	}
	if got.URL != WebMeisaiURL {
		t.Fatalf("URL = %q, want %q", got.URL, WebMeisaiURL)
	}
	if got.Title != "WEB明細" {
		t.Fatalf("Title = %q, want %q", got.Title, "WEB明細")
	}
	if got.HTML != "<html><body>statement</body></html>" {
		t.Fatalf("HTML = %q", got.HTML)
	}
}

func TestCaptureWebMeisaiReturnsWaitError(t *testing.T) {
	page := &fakePage{waitErr: errors.New("wait failed")}

	_, err := CaptureWebMeisai(page, Credentials{
		LoginID:  "member-id",
		Password: "member-pass",
	})
	if err == nil || err.Error() != "wait for mypage: wait failed" {
		t.Fatalf("expected wait error, got %v", err)
	}
}
