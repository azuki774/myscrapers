package moneyforward

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestUpdateClicksBulkUpdateAndSuica(t *testing.T) {
	dir := t.TempDir()
	cookiePath := filepath.Join(dir, "cookie.json")
	if err := os.WriteFile(cookiePath, []byte(`[{"name":"sid","value":"abc","domain":".moneyforward.com","path":"/","sameSite":"lax"}]`), 0o600); err != nil {
		t.Fatalf("write cookie: %v", err)
	}
	cookies, err := LoadCookies(cookiePath)
	if err != nil {
		t.Fatalf("LoadCookies() error = %v", err)
	}

	sess := &fakeSession{}
	if err := Update(context.Background(), UpdateOptions{
		Session: sess,
		Cookies: cookies,
	}); err != nil {
		t.Fatalf("Update() error = %v", err)
	}
	if !sess.closed {
		t.Fatalf("expected Session.Close() to be called")
	}

	want := []string{
		"Goto:https://moneyforward.com/cf",
		"ClickByXPath:/html/body/div[1]/div[2]/div[1]/div/div/div/section/p[2]/a",
		"Goto:https://moneyforward.com/",
		"ClickLinkIn:li.account.facilities-column:更新",
	}
	combined := ""
	for _, c := range sess.calls {
		combined += c + "\n"
	}
	for _, c := range sess.clicks {
		combined += c + "\n"
	}
	for _, w := range want {
		if !contains(combined, w) {
			t.Fatalf("expected call %q in trace:\n%s", w, combined)
		}
	}
	wantWaits := []time.Duration{10 * time.Second, 60 * time.Second, 10 * time.Second, 60 * time.Second}
	if len(sess.waited) != len(wantWaits) {
		t.Fatalf("waits = %v, want %v", sess.waited, wantWaits)
	}
	for i, d := range wantWaits {
		if sess.waited[i] != d {
			t.Fatalf("waits[%d] = %v, want %v", i, sess.waited[i], d)
		}
	}
}

func contains(haystack, needle string) bool {
	return len(needle) == 0 || (len(haystack) >= len(needle) && indexOf(haystack, needle) >= 0)
}

func indexOf(s, sub string) int {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return i
		}
	}
	return -1
}
