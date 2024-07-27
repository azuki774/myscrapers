package scenario

import (
	"context"
	"os"
	"path/filepath"

	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/launcher"
	"github.com/go-rod/rod/lib/proto"
	"github.com/go-rod/rod/lib/utils"
)

type testGitHub struct { // for test
	common ScenarioCommon
}

func NewTestGitHub() *testGitHub {
	return &testGitHub{
		common: ScenarioCommon{ws: os.Getenv("wsAddr"), outputDir: "/data"},
	}
}

func (t *testGitHub) Start(ctx context.Context) error {
	// browser := rod.New().MustConnect()
	// defer browser.MustClose()
	l := launcher.MustNewManaged("ws://" + t.common.ws)

	// You can also set any flag remotely before you launch the remote browser.
	// Available flags: https://peter.sh/experiments/chromium-command-line-switches
	l.Set("disable-gpu").Delete("disable-gpu")

	// Launch with headful mode
	l.Headless(true).XVFB("--server-num=5", "--server-args=-screen 0 1600x900x16")

	browser := rod.New().Client(l.MustClient()).MustConnect()
	page := browser.MustPage("https://github.com/").MustSetViewport(1920, 1080, 0, false).MustWaitLoad()
	img, err := page.Screenshot(true, &proto.PageCaptureScreenshot{
		Format: proto.PageCaptureScreenshotFormatPng,
		Clip: &proto.PageViewport{
			X:      0,
			Y:      0,
			Width:  1920,
			Height: 1080,

			Scale: 2,
		},
		CaptureBeyondViewport: true,
		OptimizeForSpeed:      true,
		FromSurface:           true,
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
