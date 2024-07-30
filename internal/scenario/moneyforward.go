package scenario

import (
	"context"
	"log/slog"
	"os"
	"path/filepath"
	"time"

	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/input"
	"github.com/go-rod/rod/lib/launcher"
)

const mfCfURL = "https://moneyforward.com/cf" // also for login page without account_selector

type ScenarioMoneyForward struct {
	common    ScenarioCommon
	browser   *rod.Browser
	user      string
	pass      string
	yyyymmdd  string
	lastMonth bool
}

func NewScenarioMoneyForward(lastMonth bool) (*ScenarioMoneyForward, error) {
	outputDir := os.Getenv("outputDir")
	user := os.Getenv("user")
	pass := os.Getenv("pass")
	yyyymm := time.Now().Format("200601")
	yyyymmdd := time.Now().Format("20060102")

	if outputDir == "" {
		outputDir = filepath.Join(defaultOutputDir, yyyymm, yyyymmdd) // /data/YYYYMM/YYYYMMDD/
	}

	if user == "" {
		slog.Error("user required")
		return nil, ErrorInvalidOption
	}
	if pass == "" {
		slog.Error("pass required")
		return nil, ErrorInvalidOption
	}

	return &ScenarioMoneyForward{
		common:    ScenarioCommon{ws: os.Getenv("wsAddr"), outputDir: outputDir},
		user:      user,
		pass:      pass,
		yyyymmdd:  yyyymmdd,
		lastMonth: lastMonth,
	}, nil
}
func (s *ScenarioMoneyForward) getBrowser(ctx context.Context) error {
	l, err := launcher.NewManaged("ws://" + s.common.ws)
	if err != nil {
		return err
	}
	// You can also set any flag remotely before you launch the remote browser.
	// Available flags: https://peter.sh/experiments/chromium-command-line-switches
	l.Set("disable-gpu").Delete("disable-gpu")

	// Launch with headful mode
	l.Headless(false).XVFB("--server-num=5", "--server-args=-screen 0 1600x900x16")

	s.browser = rod.New().Client(l.MustClient()).MustConnect()
	return nil
}

func (s *ScenarioMoneyForward) login(ctx context.Context) error {
	slog.Info("load loginpage start")
	page := s.browser.Timeout(60 * time.Second).MustPage(mfCfURL).MustWaitStable()
	slog.Info("load loginpage complete")

	slog.Info("login information input")
	// ID入力
	// loginField, err := page.Element(`[placeholder="example@moneyforward.com"]`)
	loginField, err := page.ElementX("/html/body/main/div/div/div[2]/div/section/div/form/div/div/input")
	if err != nil {
		return err
	}
	loginField.MustInput(s.user).MustType(input.Enter)

	// ここでは画面遷移しないので WaitStableしない
	time.Sleep(10 * time.Second) // 固定値スリープで代用

	// パスワード入力
	passField, err := page.ElementX("/html/body/main/div/div/div[2]/div/section/div/form/div/div[2]/input")
	// passField, err := page.Element(`[type="password"]`)
	if err != nil {
		return err
	}

	passField.MustInput(s.pass).MustType(input.Enter)

	time.Sleep(10 * time.Second) // 固定値スリープで代用
	slog.Info("login sequence complete")

	return nil
}

func (s *ScenarioMoneyForward) Start(ctx context.Context) (err error) {
	if err := s.getBrowser(ctx); err != nil {
		return err
	}
	defer s.browser.Close()

	if err := s.login(ctx); err != nil {
		slog.Error("login failed")
		return err
	}

	return nil
}
