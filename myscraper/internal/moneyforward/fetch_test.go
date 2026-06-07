package moneyforward

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

const cfNowHTML = `
<table id="cf-detail-table">
  <tr><td></td><td>12/09(月)</td><td>物販</td><td>-110</td><td>モバイルSuica</td><td>未分類</td><td>未分類</td><td></td><td></td><td></td></tr>
</table>
`

const cfLastHTML = `
<table id="cf-detail-table">
  <tr><td></td><td>11/28(木)</td><td>コーヒー</td><td>-420</td><td>三井住友</td><td>食費</td><td>喫茶</td><td></td><td></td><td></td></tr>
</table>
`

const bsHTML = `
<table class="table table-bordered">
  <thead><tr><th>月</th><th>合計</th><th>詳細</th></tr></thead>
  <tbody>
    <tr><td>2024-12</td><td>1,000,000円</td><td><a href="/d/1">詳細</a></td></tr>
  </tbody>
</table>
`

func TestFetchProducesCSVFilesAndCallsExpectedSequence(t *testing.T) {
	dir := t.TempDir()
	cookiePath := filepath.Join(dir, "cookie.json")
	if err := os.WriteFile(cookiePath, []byte(`[{"name":"sid","value":"abc","domain":".moneyforward.com","path":"/","secure":true,"httpOnly":false,"sameSite":"lax"}]`), 0o600); err != nil {
		t.Fatalf("write cookie: %v", err)
	}
	cookies, err := LoadCookies(cookiePath)
	if err != nil {
		t.Fatalf("LoadCookies() error = %v", err)
	}

	sess := &fakeSession{contentSeq: []string{cfNowHTML, cfLastHTML, bsHTML}}
	now := time.Date(2024, 12, 15, 0, 0, 0, 0, time.UTC)

	out := filepath.Join(dir, "out")
	if err := os.MkdirAll(out, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}

	if err := Fetch(context.Background(), FetchOptions{
		Session:   sess,
		Cookies:   cookies,
		OutputDir: out,
		Now:       now,
	}); err != nil {
		t.Fatalf("Fetch() error = %v", err)
	}
	if !sess.closed {
		t.Fatalf("expected Session.Close() to be called")
	}
	if len(sess.gotCookies) != 1 || sess.gotCookies[0].Name != "sid" {
		t.Fatalf("AddCookies payload = %+v", sess.gotCookies)
	}
	wantWaits := []time.Duration{cookiePrimSleep, postClickSleep, postClickSleep, postClickSleep, postClickSleep}
	if len(sess.waited) != len(wantWaits) {
		t.Fatalf("Wait calls = %v, want %v", sess.waited, wantWaits)
	}
	for i := range wantWaits {
		if sess.waited[i] != wantWaits[i] {
			t.Fatalf("Wait[%d] = %v, want %v; all waits = %v", i, sess.waited[i], wantWaits[i], sess.waited)
		}
	}

	cfData, err := os.ReadFile(filepath.Join(out, "cf.csv"))
	if err != nil {
		t.Fatalf("read cf.csv: %v", err)
	}
	if !strings.Contains(string(cfData), "2024/12/09") {
		t.Fatalf("cf.csv missing date: %q", string(cfData))
	}

	lastData, err := os.ReadFile(filepath.Join(out, "cf_lastmonth.csv"))
	if err != nil {
		t.Fatalf("read cf_lastmonth.csv: %v", err)
	}
	if !strings.Contains(string(lastData), "2024/11/28") {
		t.Fatalf("cf_lastmonth.csv missing date: %q", string(lastData))
	}

	assetData, err := os.ReadFile(filepath.Join(out, "asset_history.csv"))
	if err != nil {
		t.Fatalf("read asset_history.csv: %v", err)
	}
	if !strings.Contains(string(assetData), "1,000,000") {
		t.Fatalf("asset_history.csv missing amount: %q", string(assetData))
	}
}

func TestFetchWritesDebugHTMLWhenCurrentMonthTableIsMissing(t *testing.T) {
	dir := t.TempDir()
	cookiePath := filepath.Join(dir, "cookie.json")
	if err := os.WriteFile(cookiePath, []byte(`[{"name":"sid","value":"abc","domain":".moneyforward.com","path":"/","secure":true,"httpOnly":false,"sameSite":"lax"}]`), 0o600); err != nil {
		t.Fatalf("write cookie: %v", err)
	}
	cookies, err := LoadCookies(cookiePath)
	if err != nil {
		t.Fatalf("LoadCookies() error = %v", err)
	}

	badHTML := `<html><body><h1>not ready</h1></body></html>`
	sess := &fakeSession{contentSeq: []string{badHTML}}
	out := filepath.Join(dir, "out")
	if err := os.MkdirAll(out, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}

	err = Fetch(context.Background(), FetchOptions{
		Session:   sess,
		Cookies:   cookies,
		OutputDir: out,
		Now:       time.Date(2024, 12, 15, 0, 0, 0, 0, time.UTC),
	})
	if err == nil {
		t.Fatal("Fetch() error = nil, want parse failure")
	}
	if !strings.Contains(err.Error(), "cf_now_debug.html") {
		t.Fatalf("Fetch() error = %v, want debug html path", err)
	}
	debugHTML, readErr := os.ReadFile(filepath.Join(out, cfNowDebugFilename))
	if readErr != nil {
		t.Fatalf("read debug html: %v", readErr)
	}
	if string(debugHTML) != badHTML {
		t.Fatalf("debug html = %q, want %q", string(debugHTML), badHTML)
	}
}
