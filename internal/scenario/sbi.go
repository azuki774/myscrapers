package scenario

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"time"

	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/input"
	"github.com/go-rod/rod/lib/launcher"
	"github.com/go-rod/rod/lib/proto"
	"github.com/go-rod/rod/lib/utils"
)

const sbiLoginURL = "https://site1.sbisec.co.jp/ETGate/"

type ScenarioSBI struct {
	common  ScenarioCommon
	browser *rod.Browser
	user    string
	pass    string
	// result

	Headers []string
	Bodies  [][]string
}

func NewScenarioSBI() (*ScenarioSBI, error) {
	outputDir := os.Getenv("outputDir")
	user := os.Getenv("user")
	pass := os.Getenv("pass")

	if outputDir == "" {
		outputDir = defaultOutputDir
	}

	if user == "" {
		slog.Error("user required")
		return nil, ErrorInvalidOption
	}
	if pass == "" {
		slog.Error("pass required")
		return nil, ErrorInvalidOption
	}

	return &ScenarioSBI{
		common: ScenarioCommon{ws: os.Getenv("wsAddr"), outputDir: outputDir},
		user:   user,
		pass:   pass,
	}, nil
}

func (s *ScenarioSBI) getBrowser(ctx context.Context) error {
	l, err := launcher.NewManaged("ws://" + s.common.ws)
	if err != nil {
		return err
	}
	// You can also set any flag remotely before you launch the remote browser.
	// Available flags: https://peter.sh/experiments/chromium-command-line-switches
	l.Set("disable-gpu").Delete("disable-gpu")

	// Launch with headful mode
	l.Headless(true).XVFB("--server-num=5", "--server-args=-screen 0 1600x900x16")

	s.browser = rod.New().Client(l.MustClient()).MustConnect()
	return nil
}

func (s *ScenarioSBI) Start(ctx context.Context) error {
	if err := s.getBrowser(ctx); err != nil {
		return err
	}
	defer s.browser.Close()

	slog.Info("load loginpage start")
	page := s.browser.MustPage(sbiLoginURL).MustWaitStable()
	slog.Info("load loginpage complete")

	slog.Info("login information input")
	// ID入力
	loginField, err := page.Timeout(10 * time.Second).Element(`[name="user_id"]`)
	if err != nil {
		return err
	}
	loginField.MustInput(s.user)
	fmt.Println(loginField.MustText())

	// パスワード入力
	passField, err := page.Timeout(10 * time.Second).Element(`[name="user_password"]`)
	if err != nil {
		return err
	}
	passField.MustInput(s.pass).MustType(input.Enter) // Enter キーでそのまま送信
	page.MustWaitStable()

	page = s.browser.MustPage("https://site1.sbisec.co.jp/ETGate/?_ControlID=WPLETpfR001Control&_PageID=DefaultPID&_DataStoreID=DSWPLETpfR001Control&_ActionID=DefaultAID&getFlg=on").MustWaitStable()
	img, err := page.Screenshot(true, &proto.PageCaptureScreenshot{
		Format: proto.PageCaptureScreenshotFormatPng,
		Clip: &proto.PageViewport{
			X:      0,
			Y:      0,
			Width:  1600,
			Height: 900,

			Scale: 2,
		},
		FromSurface: true,
	})
	if err != nil {
		return err
	}
	err = utils.OutputFile(filepath.Join(s.common.outputDir, "screenshot.jpg"), img)
	if err != nil {
		return err
	}
	return nil
}
