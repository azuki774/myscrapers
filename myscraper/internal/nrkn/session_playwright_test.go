package nrkn

import (
	"context"
	"os"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/mxschmitt/playwright-go"
)

// Every URL is intercepted; this test never contacts NRKN or uses credentials.
func TestBrowserFlow(t *testing.T) {
	if os.Getenv("NRKN_BROWSER_TEST") != "1" {
		t.Skip("set NRKN_BROWSER_TEST=1 to run local Chromium fixture")
	}
	for _, scenario := range []string{"success", "different-form-name", "concurrent", "bad-login", "logout-stuck", "unexpected-asset"} {
		t.Run(scenario, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
			defer cancel()
			s, err := NewPlaywrightSession(ctx, true)
			if err != nil {
				t.Fatal(err)
			}
			defer s.Close()
			var logins, logouts, opens atomic.Int32
			loginHTML := `<div id="pc_disp"><form action="/webapp/nrk/FSrvLogon" method="post"><input name="userId"><input type="password" name="password"><input name="birthDate"><input id="btnLogin" type="submit" value="ログイン"></form></div>`
			logoutHTML := `<form name="W37S0020_Head" action="/webapp/nrk/W37S0020_View.do" method="post"></form><a href="JavaScript:document.W37S0020_Head.submit();">ログアウト</a>`
			menuHTML := logoutHTML + `<dialog id="myDialog" open><input id="btnClose" type="button" value="Close" onclick="document.getElementById('myDialog').close()"></dialog><form name="W37S1040_Form" action="/webapp/nrk/W37S1040_AssetValuePlan.do" method="post"></form><a href="JavaScript:document.W37S1040_Form.submit();">資産評価額照会</a>`
			if scenario == "different-form-name" {
				menuHTML = strings.ReplaceAll(menuHTML, "W37S1040_Form", "DifferentAssetForm")
				menuHTML = strings.ReplaceAll(menuHTML, "資産評価額照会</a>", "資産評価額照会<span>現在の資産状況を確認できます</span></a>")
			}
			err = s.context.Route("**/*", func(route playwright.Route) {
				u := route.Request().URL()
				body := ""
				switch {
				case strings.HasSuffix(u, "/FSrvLogon"):
					n := logins.Add(1)
					if scenario == "bad-login" {
						body = loginHTML + "認証に失敗しました"
					} else if scenario == "concurrent" && n == 1 {
						body = loginHTML + "990003"
					} else {
						_ = route.Fulfill(playwright.RouteFulfillOptions{ContentType: playwright.String("text/html"), Body: `<script>location.replace('/webapp/nrk/W37S0030_View.do')</script>`})
						return
					}
				case strings.HasSuffix(u, "/W37S0030_View.do"):
					body = menuHTML
				case strings.HasSuffix(u, "/W37S1040_AssetValuePlan.do"):
					opens.Add(1)
					if scenario == "unexpected-asset" {
						body = logoutHTML + "一時的なエラー"
					} else {
						body = logoutHTML + `<div id="W37S104001">資産評価額</div>`
					}
				case strings.HasSuffix(u, "/W37S0020_View.do"):
					logouts.Add(1)
					if scenario == "logout-stuck" {
						body = logoutHTML + "資産評価額"
					} else {
						body = "ご利用ありがとうございました"
					}
				default:
					body = loginHTML
				}
				_ = route.Fulfill(playwright.RouteFulfillOptions{ContentType: playwright.String("text/html; charset=utf-8"), Body: body})
			})
			if err != nil {
				t.Fatal(err)
			}
			err = s.Login(ctx, "dummy-id", "dummy-password", "20000101")
			if scenario == "concurrent" {
				if err != ErrConcurrentLogin {
					t.Fatalf("first login = %v", err)
				}
				err = s.Login(ctx, "dummy-id", "dummy-password", "20000101")
			}
			if scenario == "bad-login" {
				if err == nil {
					t.Fatal("bad login succeeded")
				}
				if e := s.Login(ctx, "dummy-id", "dummy-password", "20000101"); e == nil {
					t.Fatal("unapproved retry")
				}
				if e := s.Logout(ctx); e != nil {
					t.Fatal(e)
				}
				if logins.Load() != 1 || logouts.Load() != 0 {
					t.Fatal("unexpected requests")
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			navCtx := ctx
			if scenario == "unexpected-asset" {
				var c context.CancelFunc
				navCtx, c = context.WithTimeout(ctx, time.Second)
				defer c()
			}
			err = s.NavigateToAssets(navCtx)
			if scenario == "unexpected-asset" {
				if err == nil {
					t.Fatal("accepted missing asset content")
				}
			} else if err != nil {
				t.Fatal(err)
			}
			err = s.Logout(ctx)
			if scenario == "logout-stuck" {
				if err == nil {
					t.Fatal("accepted still-authenticated logout")
				}
			} else if err != nil {
				t.Fatal(err)
			}
			if logouts.Load() != 1 || opens.Load() != 1 {
				t.Fatalf("asset=%d logout=%d", opens.Load(), logouts.Load())
			}
			expected := int32(1)
			if scenario == "concurrent" {
				expected = 2
			}
			if logins.Load() != expected {
				t.Fatalf("login count %d", logins.Load())
			}
			if err = s.Logout(ctx); err == nil {
				t.Fatal("second logout allowed")
			}
			if logouts.Load() != 1 {
				t.Fatal("logout retried")
			}
		})
	}
}

// Diagnose failures without exposing page content, URLs or Playwright call logs.
func TestBrowserNavigationDiagnostics(t *testing.T) {
	if os.Getenv("NRKN_BROWSER_TEST") != "1" {
		t.Skip("set NRKN_BROWSER_TEST=1 to run local Chromium fixture")
	}
	for _, tc := range []struct {
		name, html, want string
	}{
		{"missing-link", `<p>private-account-marker</p>`, "stage=click, matching_elements=0, screen=menu"},
		{"blocked-link", `<a href="#">資産評価額照会</a><div style="position:fixed;inset:0;z-index:100">private-account-marker</div>`, "stage=click, matching_elements=1, screen=menu"},
		{"no-navigation", `<a href="javascript:void(0)">資産評価額照会</a>`, "stage=navigation, matching_elements=1, screen=menu"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			s, err := NewPlaywrightSession(context.Background(), true)
			if err != nil {
				t.Fatal(err)
			}
			defer s.Close()
			if err := s.context.Route("**/*", func(route playwright.Route) {
				_ = route.Fulfill(playwright.RouteFulfillOptions{ContentType: playwright.String("text/html; charset=utf-8"), Body: tc.html})
			}); err != nil {
				t.Fatal(err)
			}
			if _, err := s.page.Goto("https://www.nrkn.co.jp/webapp/nrk/W37S0030_View.do?token=private-token-marker"); err != nil {
				t.Fatal(err)
			}
			ctx, cancel := context.WithTimeout(context.Background(), time.Second)
			defer cancel()
			err = s.NavigateToAssets(ctx)
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("navigation error = %v, want %q", err, tc.want)
			}
			for _, secret := range []string{"private-account-marker", "private-token-marker", "https://", "Call log:"} {
				if strings.Contains(err.Error(), secret) {
					t.Fatal("navigation error exposed private diagnostic data")
				}
			}
		})
	}
}
