package cli

import "testing"

func TestParseArgs(t *testing.T) {
	t.Run("requires url", func(t *testing.T) {
		_, err := ParseArgs([]string{"--out", "tmp/page.html"})
		if err == nil || err.Error() != "--url is required" {
			t.Fatalf("expected url validation error, got %v", err)
		}
	})

	t.Run("applies defaults", func(t *testing.T) {
		got, err := ParseArgs([]string{"--url", "https://github.com"})
		if err != nil {
			t.Fatalf("ParseArgs() error = %v", err)
		}
		if got.URL != "https://github.com" {
			t.Fatalf("URL = %q, want %q", got.URL, "https://github.com")
		}
		if got.OutputPath != "tmp/page.html" {
			t.Fatalf("OutputPath = %q, want %q", got.OutputPath, "tmp/page.html")
		}
		if !got.Headless {
			t.Fatalf("Headless = %v, want true", got.Headless)
		}
	})
}
