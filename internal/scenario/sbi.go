package scenario

import (
	"context"
	"fmt"
	"log/slog"
	"myscrapers/internal/csv"
	"os"
	"strings"
	"time"

	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/launcher"
)

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
		common: ScenarioCommon{ws: os.Getenv("wsAddr"), outputDir: "/data"},
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
	loginUrl := "https://site1.sbisec.co.jp/ETGate/?_ControlID=WPLETmgR001Control&_PageID=WPLETmgR001Mdtl20&_DataStoreID=DSWPLETmgR001Control&_ActionID=DefaultAID&burl=iris_top&cat1=market&cat2=top&dir=tl1-top%7Ctl2-map%7Ctl5-jpn&file=index.html&getFlg=on"
	if err := s.getBrowser(ctx); err != nil {
		return err
	}
	defer s.browser.Close()

	slog.Info("load login page start")
	page := s.browser.MustPage(loginUrl).MustWaitStable()
	slog.Info("load login page complete")

	el, err := page.ElementX("//*[@id='market_top_pain']/div[6]/div[2]/table[1]")
	if err != nil {
		return err
	}

	slog.Info("get elements start")

	// header
	items, err := el.Elements("th")
	if err != nil {
		return err
	}

	for _, item := range items {
		t, err := item.Timeout(10 * time.Second).Text()
		if err != nil {
			return err
		}
		s.Headers = append(s.Headers, t)
	}

	// body
	items, err = el.Elements("td")
	if err != nil {
		return err
	}

	var tmpBody []string
	for _, item := range items {
		t, err := item.Timeout(10 * time.Second).Text()
		if err != nil {
			return err
		}
		// 改行は半角スペースに
		t = strings.ReplaceAll(t, "\n", " ")
		tmpBody = append(tmpBody, t)
	}

	headerLen := len(s.Headers)
	BodiesLen := len(tmpBody) / len(s.Headers) // TODO: validation
	for i := 0; i < BodiesLen; i++ {
		var insertBody []string
		for j := 0; j < headerLen; j++ {
			insertBody = append(insertBody, tmpBody[i*headerLen+j])
		}
		s.Bodies = append(s.Bodies, insertBody)
	}

	slog.Info("get elements complete")
	fmt.Println(s.Headers)
	fmt.Println("-----------------------------")
	fmt.Println(s.Bodies)

	csv.WriteFile("./output.csv", s.Headers, s.Bodies)
	return nil
}
