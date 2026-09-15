package cli

import (
	"bytes"
	"context"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/azuki774/myscrapers/myscraper/internal/nrkn"
)

type fakeNRKNRunner struct{}

func (fakeNRKNRunner) RunAssets(context.Context, nrkn.FetchOptions) error { return nil }

type fakeNRKNS3 struct {
	key  string
	body []byte
}

func (f *fakeNRKNS3) KeyForTime(time.Time) string { return f.key }
func (f *fakeNRKNS3) PutJSON(_ context.Context, _ string, r io.Reader) error {
	var err error
	f.body, err = io.ReadAll(r)
	return err
}

func TestRunNRKNRequiresCredentials(t *testing.T) {
	t.Setenv("NRKN_ID", "")
	t.Setenv("NRKN_PASS", "")
	t.Setenv("NRKN_BIRTHDAY", "")
	var out, errOut bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&errOut, nil))
	if got := RunNRKN([]string{"nrkn"}, &out, &errOut, logger, fakeNRKNRunner{}); got != 2 {
		t.Fatalf("exit=%d, want 2", got)
	}
	if out.Len() != 0 {
		t.Fatalf("stdout must remain clean: %q", out.String())
	}
}

func TestWriteNRKNJSONAtomic0600(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "assets.json")
	raw, err := WriteNRKNJSON(&bytes.Buffer{}, path, &nrkn.Assets{FetchedAt: time.Unix(0, 0), Status: nrkn.StatusOK})
	if err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(raw, got) {
		t.Fatal("file differs from returned bytes")
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0600 {
		t.Fatalf("mode=%o", info.Mode().Perm())
	}
}

func TestUploadNRKNJSONPreservesEmittedBytes(t *testing.T) {
	var emitted bytes.Buffer
	a := &nrkn.Assets{FetchedAt: time.Unix(0, 0), Status: nrkn.StatusOK,
		Holdings: []nrkn.Holding{{ProductCode: "09999", CompositeFIGI: "BBGTESTFUND"}}}
	raw, err := WriteNRKNJSON(&emitted, "", a)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains(raw, []byte(`"composite_figi": "BBGTESTFUND"`)) {
		t.Fatal("emitted JSON is missing composite FIGI")
	}
	store := &fakeNRKNS3{key: "nrkn/2026/09/example.json"}
	key, err := UploadNRKNJSON(context.Background(), store, time.Unix(0, 0), raw)
	if err != nil {
		t.Fatal(err)
	}
	if key != store.key || !bytes.Equal(raw, store.body) || !bytes.Equal(raw, emitted.Bytes()) {
		t.Fatalf("S3 payload differs")
	}
}
