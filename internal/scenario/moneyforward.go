package scenario

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"time"

	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/launcher"
)

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
	l.Headless(true).XVFB("--server-num=5", "--server-args=-screen 0 1600x900x16")

	s.browser = rod.New().Client(l.MustClient()).MustConnect()
	return nil
}
func (s *ScenarioMoneyForward) Start(ctx context.Context) (err error) {
	if err := s.getBrowser(ctx); err != nil {
		return err
	}
	defer s.browser.Close()

	fmt.Println("TODO")

	return nil
}
