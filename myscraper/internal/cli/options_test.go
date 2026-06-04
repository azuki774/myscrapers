package cli

import (
	"testing"

	"github.com/azuki774/myscrapers/myscraper/internal/smbccard"
)

func TestParseArgs(t *testing.T) {
	t.Run("requires url for fetch mode", func(t *testing.T) {
		_, err := ParseArgs([]string{"--mode", ModeFetchURL})
		if err == nil || err.Error() != "--url is required when --mode=fetch-url" {
			t.Fatalf("expected url validation error, got %v", err)
		}
	})

	t.Run("applies fetch defaults", func(t *testing.T) {
		got, err := ParseArgs([]string{"--url", "https://github.com"})
		if err != nil {
			t.Fatalf("ParseArgs() error = %v", err)
		}
		if got.Mode != ModeFetchURL {
			t.Fatalf("Mode = %q, want %q", got.Mode, ModeFetchURL)
		}
		if got.OutputPath != "tmp/page.html" {
			t.Fatalf("OutputPath = %q, want %q", got.OutputPath, "tmp/page.html")
		}
		if !got.Headless {
			t.Fatalf("Headless = %v, want true", got.Headless)
		}
	})

	t.Run("applies smbc defaults without url", func(t *testing.T) {
		got, err := ParseArgs([]string{"--mode", smbccard.ModeWebMeisaiHTMLDump})
		if err != nil {
			t.Fatalf("ParseArgs() error = %v", err)
		}
		if got.Mode != smbccard.ModeWebMeisaiHTMLDump {
			t.Fatalf("Mode = %q, want %q", got.Mode, smbccard.ModeWebMeisaiHTMLDump)
		}
		if got.URL != "" {
			t.Fatalf("URL = %q, want empty", got.URL)
		}
		if got.OutputPath != "tmp/smbccard-webmeisai.html" {
			t.Fatalf("OutputPath = %q, want %q", got.OutputPath, "tmp/smbccard-webmeisai.html")
		}
	})

	t.Run("rejects unknown mode", func(t *testing.T) {
		_, err := ParseArgs([]string{"--mode", "unknown"})
		if err == nil || err.Error() != "unsupported --mode: unknown" {
			t.Fatalf("expected mode validation error, got %v", err)
		}
	})
}
