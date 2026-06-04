package scrape

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

type fakeBrowser struct {
	snapshot PageSnapshot
	err      error
}

func (f fakeBrowser) Fetch(ctx context.Context, url string, headless bool) (PageSnapshot, error) {
	return f.snapshot, f.err
}

func TestServiceRunWritesHTML(t *testing.T) {
	out := filepath.Join(t.TempDir(), "page.html")
	svc := Service{
		Browser: fakeBrowser{
			snapshot: PageSnapshot{
				URL:   "https://github.com",
				Title: "GitHub",
				HTML:  "<html><body><h1>GitHub</h1></body></html>",
			},
		},
	}

	result, err := svc.Run(context.Background(), Request{
		URL:        "https://github.com",
		OutputPath: out,
		Headless:   true,
	})
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if result.Title != "GitHub" {
		t.Fatalf("Title = %q, want %q", result.Title, "GitHub")
	}

	data, err := os.ReadFile(out)
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}
	if string(data) != "<html><body><h1>GitHub</h1></body></html>" {
		t.Fatalf("HTML = %q", string(data))
	}
}
