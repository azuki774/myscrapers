// Package chromium locates a Chromium-compatible browser binary on the
// host PATH. The dev shell (Nix) provides `chromium`; Debian-based
// Docker images install `google-chrome-stable`; macOS typically
// installs `google-chrome` via brew cask or `/Applications`.
package chromium

import (
	"fmt"
	"os/exec"
)

// ExecutablePath returns the first chromium-compatible binary found in
// PATH, searched in the order chromium, chromium-browser, google-chrome,
// google-chrome-stable. Returns an error if none is found so callers can
// surface a clear "install chromium" message instead of letting
// Playwright fail later with an opaque launch error.
func ExecutablePath() (string, error) {
	for _, candidate := range []string{"chromium", "chromium-browser", "google-chrome", "google-chrome-stable"} {
		if path, err := exec.LookPath(candidate); err == nil {
			return path, nil
		}
	}
	return "", fmt.Errorf("no chromium-compatible browser found in PATH")
}
