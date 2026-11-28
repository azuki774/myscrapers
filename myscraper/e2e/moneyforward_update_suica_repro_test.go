package e2e

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/azuki774/myscrapers/myscraper/internal/moneyforward"
)

// suicaHomeFixture mimics the MoneyForward home page account list
// (https://moneyforward.com/) as observed in the production warning:
// every row is `li.account.facilities-column`, rows that finished or
// have not started refreshing contain an <a>更新</a> link, rows that the
// just-pressed 一括更新 already put into the loading state contain a
// nested <li class="loding">更新中</li>. Ten elements in total match
// getByText('更新'), exactly like the logged strict-mode violation.
//
// The Suica row is first, as it is on the real page, and its link
// targets /update/suica so the test can tell which link was pressed.
const suicaHomeFixture = `<!doctype html>
<html lang="ja"><body>
<ul id="account-list">
  <li class="account facilities-column">
    <a href="/accounts/suica">モバイルSuica</a>
    <a rel="nofollow" data-remote="true" data-method="post" href="/update/suica">更新</a>
  </li>
  <li class="account facilities-column">
    <a href="/accounts/a">三井住友銀行</a>
    <a rel="nofollow" data-remote="true" data-method="post" href="/update/a">更新</a>
  </li>
  <li class="account facilities-column">
    <a href="/accounts/b">楽天銀行</a>
    <div class="status"><ul><li class="loding lIPdYqzA5FVVESiRtKI6dFHt5c">更新中</li></ul></div>
  </li>
  <li class="account facilities-column">
    <a href="/accounts/c">住信SBIネット銀行</a>
    <a rel="nofollow" data-remote="true" data-method="post" href="/update/c">更新</a>
  </li>
  <li class="account facilities-column">
    <a href="/accounts/d">PayPay</a>
    <div class="status"><ul><li class="loding 9zR_MWC4Ezq5vBoj9l5-bpGYORNNFX3iHadHVOHK_8M">更新中</li></ul></div>
  </li>
  <li class="account facilities-column">
    <a href="/accounts/e">Amazon</a>
    <a rel="nofollow" data-remote="true" data-method="post" href="/update/e">更新</a>
  </li>
  <li class="account facilities-column">
    <a href="/accounts/f">楽天カード</a>
    <div class="status"><ul><li class="loding aaa_MWC4Ezq5vBoj9l5-bpGYORNNFX3iHadHVOHK_8M">更新中</li></ul></div>
  </li>
  <li class="account facilities-column">
    <a href="/accounts/g">dカード</a>
    <a rel="nofollow" data-remote="true" data-method="post" href="/update/g">更新</a>
  </li>
  <li class="account facilities-column">
    <a href="/accounts/h">SBI証券</a>
    <div class="status"><ul><li class="loding bbb_MWC4Ezq5vBoj9l5-bpGYORNNFX3iHadHVOHK_8M">更新中</li></ul></div>
  </li>
  <li class="account facilities-column">
    <a href="/accounts/i">セブン銀行</a>
    <a rel="nofollow" data-remote="true" data-method="post" href="/update/i">更新</a>
  </li>
</ul>
</body></html>`

// suicaUpdatingFixture is the same 10-row page with the モバイルSuica row
// already refreshing: its 更新 link is replaced by a 更新中 loading li,
// i.e. the state the bulk press leaves the row in.
var suicaUpdatingFixture = strings.Replace(suicaHomeFixture,
	`<a rel="nofollow" data-remote="true" data-method="post" href="/update/suica">更新</a>`,
	`<div class="status"><ul><li class="loding suicaLoading1234">更新中</li></ul></div>`, 1)

// serveSuicaHome serves body for every path and reports the path of any
// account "更新" link the browser actually followed.
func serveSuicaHome(t *testing.T, body string) (*httptest.Server, chan string) {
	t.Helper()
	clicked := make(chan string, 4)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.URL.Path, "/update/") {
			select {
			case clicked <- r.URL.Path:
			default:
			}
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = io.WriteString(w, body)
	}))
	t.Cleanup(srv.Close)
	return srv, clicked
}

// TestUpdateSuicaStepPressesSuicaLink checks the production "--update"
// Suica step against a saved-shape fixture using the production
// PlaywrightSession: scope to the li.account.facilities-column row
// containing モバイルSuica and click that row's exact-text 更新 link.
// The 更新中 loading rows and the other accounts' 更新 links must not
// match; /update/suica is the only navigation the step may cause.
func TestUpdateSuicaStepPressesSuicaLink(t *testing.T) {
	if os.Getenv("PLAYWRIGHT_E2E") != "1" {
		t.Skip("set PLAYWRIGHT_E2E=1 to run browser test")
	}
	srv, clicked := serveSuicaHome(t, suicaHomeFixture)
	ctx := context.Background()

	sess, err := moneyforward.NewPlaywrightSession(ctx, true)
	if err != nil {
		t.Fatalf("NewPlaywrightSession() error = %v", err)
	}
	defer sess.Close()

	if err := sess.Goto(ctx, srv.URL); err != nil {
		t.Fatalf("Goto() error = %v", err)
	}
	if err := sess.ClickLinkInRow(ctx, "li.account.facilities-column", "モバイルSuica", "更新"); err != nil {
		t.Fatalf("ClickLinkInRow() error = %v", err)
	}

	select {
	case path := <-clicked:
		if path != "/update/suica" {
			t.Fatalf("pressed %q, want /update/suica (the モバイルSuica row)", path)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("no 更新 link was followed")
	}
}

// TestUpdateSuicaStepRowMissingFailsFast checks the negative case: when
// the モバイルSuica row has no 更新 link (only a 更新中 loading li, as
// right after the bulk press), ClickLinkInRow must return an error
// naming the "row missing or already updating" condition promptly —
// well under Playwright's default 30s click timeout — instead of
// timing out on a click that can never happen.
func TestUpdateSuicaStepRowMissingFailsFast(t *testing.T) {
	if os.Getenv("PLAYWRIGHT_E2E") != "1" {
		t.Skip("set PLAYWRIGHT_E2E=1 to run browser test")
	}
	srv, _ := serveSuicaHome(t, suicaUpdatingFixture)
	ctx := context.Background()

	sess, err := moneyforward.NewPlaywrightSession(ctx, true)
	if err != nil {
		t.Fatalf("NewPlaywrightSession() error = %v", err)
	}
	defer sess.Close()

	if err := sess.Goto(ctx, srv.URL); err != nil {
		t.Fatalf("Goto() error = %v", err)
	}
	start := time.Now()
	err = sess.ClickLinkInRow(ctx, "li.account.facilities-column", "モバイルSuica", "更新")
	elapsed := time.Since(start)
	if err == nil {
		t.Fatal("ClickLinkInRow() error = nil, want row-missing error")
	}
	if !strings.Contains(err.Error(), "row missing or already updating") {
		t.Fatalf("ClickLinkInRow() error = %v, want it to contain \"row missing or already updating\"", err)
	}
	if elapsed >= 10*time.Second {
		t.Fatalf("ClickLinkInRow() took %v, want well under the 30s click timeout", elapsed)
	}
}
