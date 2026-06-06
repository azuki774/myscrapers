package moneyforward

import (
	"context"
	"os"
	"path/filepath"
	"sort"
	"sync"
	"testing"
)

type fakeUploader struct {
	mu     sync.Mutex
	keys   []string
	prefix string
}

func (f *fakeUploader) Upload(_ context.Context, path string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.keys = append(f.keys, f.prefix+"/"+filepath.Base(path))
	return nil
}

func TestFetchOptionallyUploads(t *testing.T) {
	dir := t.TempDir()
	cookiePath := filepath.Join(dir, "cookie.json")
	if err := os.WriteFile(cookiePath, []byte(`[{"name":"sid","value":"abc","domain":".moneyforward.com","path":"/","sameSite":"lax"}]`), 0o600); err != nil {
		t.Fatalf("write cookie: %v", err)
	}
	cookies, err := LoadCookies(cookiePath)
	if err != nil {
		t.Fatalf("LoadCookies() error = %v", err)
	}

	sess := &fakeSession{contentSeq: []string{cfNowHTML, cfLastHTML, bsHTML}}
	upl := &fakeUploader{prefix: "myscrapers/moneyforward"}

	out := filepath.Join(dir, "out")
	if err := os.MkdirAll(out, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}

	if err := Fetch(context.Background(), FetchOptions{
		Session:   sess,
		Cookies:   cookies,
		OutputDir: out,
		Uploader:  upl,
	}); err != nil {
		t.Fatalf("Fetch() error = %v", err)
	}

	sort.Strings(upl.keys)
	want := []string{
		"myscrapers/moneyforward/asset_history.csv",
		"myscrapers/moneyforward/cf.csv",
		"myscrapers/moneyforward/cf_lastmonth.csv",
	}
	if len(upl.keys) != len(want) {
		t.Fatalf("keys = %v, want %v", upl.keys, want)
	}
	for i, w := range want {
		if upl.keys[i] != w {
			t.Fatalf("keys[%d] = %q, want %q", i, upl.keys[i], w)
		}
	}
}
