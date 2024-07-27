package scenario

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/launcher"
	"github.com/go-rod/rod/lib/proto"
	"github.com/go-rod/rod/lib/utils"
)

const defaultOutputDir = "/data"

type testGitHub struct { // for test
	common  ScenarioCommon
	browser *rod.Browser
}

func NewTestGitHub() *testGitHub {
	outputDir := os.Getenv("outputDir")
	if outputDir == "" {
		outputDir = defaultOutputDir
	}
	return &testGitHub{
		common: ScenarioCommon{ws: os.Getenv("wsAddr"), outputDir: outputDir},
	}
}
func (t *testGitHub) getBrowser(ctx context.Context) {
	l := launcher.MustNewManaged("ws://" + t.common.ws)

	// You can also set any flag remotely before you launch the remote browser.
	// Available flags: https://peter.sh/experiments/chromium-command-line-switches
	l.Set("disable-gpu").Delete("disable-gpu")

	// Launch with headful mode
	l.Headless(false).XVFB("--server-num=5", "--server-args=-screen 0 1600x900x16")

	t.browser = rod.New().Client(l.MustClient()).MustConnect()
}

func (t *testGitHub) Start(ctx context.Context) error {
	defer func() {
		if rec := recover(); rec != nil {
			slog.Error("panic occured", "err", fmt.Errorf("%v", rec))
		}
	}()
	t.getBrowser(ctx)
	defer t.browser.MustClose()

	page := t.browser.MustPage("https://github.com/").MustSetViewport(1920, 1080, 0, false).MustWaitLoad()
	img, err := page.Screenshot(true, &proto.PageCaptureScreenshot{
		Format: proto.PageCaptureScreenshotFormatPng,
		Clip: &proto.PageViewport{
			X:      0,
			Y:      0,
			Width:  1280,
			Height: 720,

			Scale: 2,
		},
		FromSurface: true,
	})
	if err != nil {
		return err
	}
	err = utils.OutputFile(filepath.Join(t.common.outputDir, "screenshot.jpg"), img)
	if err != nil {
		return err
	}
	return nil
}
