package importer

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"strings"
	"time"

	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/launcher"
)

type ImporterCF struct {
	common      ImporterCommon
	browser     *rod.Browser
	yyyymmdd    string
	inputCFFile string
}

func NewImporterCF(ctx context.Context) (*ImporterCF, error) {
	outputDir := os.Getenv("outputDir")
	yyyymmdd := time.Now().Format("20060102")

	if outputDir == "" {
		outputDir = defaultOutputDir
	}

	return &ImporterCF{
		common:      ImporterCommon{ws: os.Getenv("wsAddr"), outputDir: outputDir},
		yyyymmdd:    yyyymmdd,
		inputCFFile: "file:///data/cf.html", // TODO
	}, nil
}

func (i *ImporterCF) getBrowser(ctx context.Context) error {
	l, err := launcher.NewManaged("ws://" + i.common.ws)
	if err != nil {
		return err
	}
	// You can also set any flag remotely before you launch the remote browser.
	// Available flags: https://peter.sh/experiments/chromium-command-line-switches
	l.Set("disable-gpu").Delete("disable-gpu")

	// Launch with headful mode
	l.Headless(true).XVFB("--server-num=5", "--server-args=-screen 0 1600x900x16")

	i.browser = rod.New().Client(l.MustClient()).MustConnect()
	return nil
}
func (i *ImporterCF) Start(ctx context.Context) (err error) {
	if err := i.getBrowser(ctx); err != nil {
		slog.Error("failed to get browser")
		return err
	}

	page := i.browser.MustPage(i.inputCFFile).MustWaitStable()
	cfDetailTable, err := page.Element(`[id=cf-detail-table]`)
	if err != nil {
		slog.Error("failed to get cfDetailTable")
		return err
	}

	ths := cfDetailTable.MustElements("th")

	var header []string
	var bodies [][]string

	for _, th := range ths {
		// セレクターの選択肢のテキストを消す
		txt := strings.Split(th.MustText(), " ")[0]
		// 改行を消す
		txt = strings.ReplaceAll(txt, "\n", "")
		// 無駄な空白を消す
		txt = strings.ReplaceAll(txt, " ", "")
		header = append(header, txt)
	}
	fmt.Println(len(header))
	fmt.Println(header)

	recordRows, err := cfDetailTable.Elements(`[class="transaction_list js-cf-edit-container target-active"`)
	if err != nil {
		slog.Error("failed to get recordRows")
		return err
	}

	for _, recordRow := range recordRows {
		var row []string
		spans := recordRow.MustElements("td")
		for _, span := range spans {
			// セレクターの選択肢のテキストを消す
			txt := strings.Split(span.MustText(), " ")[0]
			// 改行を消す
			txt = strings.ReplaceAll(txt, "\n", "")
			// 無駄な空白を消す
			txt = strings.ReplaceAll(txt, " ", "")
			row = append(row, txt)
		}
		bodies = append(bodies, row)
		fmt.Println(len(row))
	}
	fmt.Println(bodies)

	return nil
}
