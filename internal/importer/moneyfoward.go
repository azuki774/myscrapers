package importer

import (
	"context"
	"log/slog"
	"myscrapers/internal/csv"
	"os"
	"path/filepath"
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

// cfPage は /cf のページを rod で取得したもの
func (i *ImporterCF) getHeader(ctx context.Context, cfPage *rod.Page) (header []string, err error) {
	cfDetailTable, err := cfPage.Element(`[id=cf-detail-table]`)
	if err != nil {
		slog.Error("failed to get cf-detail-table")
		return []string{}, err
	}
	ths := cfDetailTable.MustElements("th")

	if err != nil {
		slog.Error("failed to get cfDetailTable")
		return []string{}, err
	}

	for _, th := range ths {
		// セレクターの選択肢のテキストを消す
		txt := strings.Split(th.MustText(), " ")[0]
		// 改行を消す
		txt = strings.ReplaceAll(txt, "\n", "")
		// 無駄な空白を消す
		txt = strings.ReplaceAll(txt, " ", "")
		header = append(header, txt)
	}

	return header, nil
}

// cfPage は /cf のページを rod で取得したもの
func (i *ImporterCF) getBody(ctx context.Context, cfPage *rod.Page) (bodies [][]string, err error) {
	cfDetailTable, err := cfPage.Element(`[id=cf-detail-table]`)
	if err != nil {
		slog.Error("failed to get cf-detail-table")
		return [][]string{}, err
	}

	recordRows, err := cfDetailTable.Elements(`[class="transaction_list js-cf-edit-container target-active"`) // 1行ごとのrecordsのフィールドを特定する
	if err != nil {
		slog.Error("failed to get recordRows")
		return [][]string{}, err
	}

	for _, recordRow := range recordRows {
		var row []string
		spans := recordRow.MustElements("td") // 1行ごとのレコードから各セルを抽出
		for _, span := range spans {
			// セレクターの選択肢のテキストを消す
			txt := strings.Split(span.MustText(), " ")[0]
			// 改行を消す
			txt = strings.ReplaceAll(txt, "\n", "")
			// 無駄な空白を消さない
			row = append(row, txt)
		}
		bodies = append(bodies, row)
	}

	return bodies, nil
}

func (i *ImporterCF) Start(ctx context.Context) (err error) {
	if err := i.getBrowser(ctx); err != nil {
		slog.Error("failed to get browser")
		return err
	}

	page := i.browser.MustPage(i.inputCFFile).MustWaitStable()

	var header []string
	var bodies [][]string

	header, err = i.getHeader(ctx, page)
	if err != nil {
		slog.Error("failed to get header")
		return err
	}

	bodies, err = i.getBody(ctx, page)
	if err != nil {
		slog.Error("failed to get bodies")
		return err
	}

	// validation
	if err := csv.ValidateCF(header, bodies); err != nil {
		return err
	}

	// csv書き込み
	if err := csv.WriteFile(filepath.Join(i.common.outputDir, "output.csv"), header, bodies); err != nil {
		slog.Error("failed to output csv")
		return err
	}
	slog.Info("output csv complete")
	return nil
}
