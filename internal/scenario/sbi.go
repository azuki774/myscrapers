package scenario

import (
	"context"
	"fmt"
	"log/slog"
	"myscrapers/internal/csv"
	"os"
	"path/filepath"
	"time"

	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/input"
	"github.com/go-rod/rod/lib/launcher"
	"github.com/go-rod/rod/lib/proto"
)

const sbiLoginURL = "https://site1.sbisec.co.jp/ETGate/"
const portfolioURL = "https://site1.sbisec.co.jp/ETGate/?_ControlID=WPLETpfR001Control&_PageID=DefaultPID&_DataStoreID=DSWPLETpfR001Control&_ActionID=DefaultAID&getFlg=on"

const portfolioFieldSize = 12

type ScenarioSBI struct {
	common   ScenarioCommon
	browser  *rod.Browser
	user     string
	pass     string
	yyyymmdd string
}

func NewScenarioSBI() (*ScenarioSBI, error) {
	outputDir := os.Getenv("outputDir")
	user := os.Getenv("user")
	pass := os.Getenv("pass")
	yyyymmdd := time.Now().Format("20060102")

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
		common:   ScenarioCommon{ws: os.Getenv("wsAddr"), outputDir: outputDir},
		user:     user,
		pass:     pass,
		yyyymmdd: yyyymmdd,
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

func (s *ScenarioSBI) login(ctx context.Context) error {
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

	// パスワード入力
	passField, err := page.Timeout(10 * time.Second).Element(`[name="user_password"]`)
	if err != nil {
		return err
	}
	passField.MustInput(s.pass).MustType(input.Enter) // Enter キーでそのまま送信
	if err := page.WaitStable(10 * time.Second); err != nil {
		return err
	}
	slog.Info("login sequence complete")
	return nil
}

func (s *ScenarioSBI) getPortfolio(ctx context.Context) error {
	// ポートフォリオページに移動
	slog.Info("move Portfolio page")
	page := s.browser.MustPage(portfolioURL).MustWaitStable()
	_, err := page.Screenshot(true, &proto.PageCaptureScreenshot{
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
	// err = utils.OutputFile(filepath.Join(s.common.outputDir, fmt.Sprintf("%s_pic.jpg", s.yyyymmdd)), img)
	// if err != nil {
	// 	return err
	// }

	tables1 := page.MustElements("tbody") // table1: tbody を全部抜き出し
	var tables2 []rod.Elements            // table2: ポートフォリオテーブルのみ抜き出し
	for _, t := range tables1 {
		val, err := t.Elements("td")
		if err != nil {
			continue
		}
		if len(val) == 0 {
			continue
		}
		if val[0].MustText() != "取引" {
			// 一番左上の要素が「取引」でないものは目的のテーブルでないので除外
			continue
		}
		slog.Info("datected Portfolio table")
		tables2 = append(tables2, val)
	}

	if len(tables2) == 0 {
		// 1つもテーブルが見つからなかった場合、ログインに失敗していそう
		slog.Warn("portfolio table not found. perhaps login not sucessfully")
		return fmt.Errorf("portfolio table not found")
	}

	// 1テーブルごとに処理
	for i, table := range tables2 {
		elNum := len(table)
		rowNum := elNum / portfolioFieldSize
		slog.Info("extract table", "table", i, "elementNum", elNum, "rowNum", rowNum)
		// 取引,ファンド名,買付日,数量,取得単価,現在値,前日比,前日比（％）,損益,損益（％）,評価額,編集
		var headers []string
		var bodies [][]string

		headers = []string{
			table[0].MustText(),
			table[1].MustText(),
			table[2].MustText(),
			table[3].MustText(),
			table[4].MustText(),
			table[5].MustText(),
			table[6].MustText(),
			table[7].MustText(),
			table[8].MustText(),
			table[9].MustText(),
			table[10].MustText(),
			table[11].MustText(),
		}
		for i := 1; i < rowNum; i++ { // Body (not Header)
			tmpBody := []string{
				table[i*portfolioFieldSize+0].MustText(),
				table[i*portfolioFieldSize+1].MustText(),
				table[i*portfolioFieldSize+2].MustText(),
				table[i*portfolioFieldSize+3].MustText(),
				table[i*portfolioFieldSize+4].MustText(),
				table[i*portfolioFieldSize+5].MustText(),
				table[i*portfolioFieldSize+6].MustText(),
				table[i*portfolioFieldSize+7].MustText(),
				table[i*portfolioFieldSize+8].MustText(),
				table[i*portfolioFieldSize+9].MustText(),
				table[i*portfolioFieldSize+10].MustText(),
				table[i*portfolioFieldSize+11].MustText(),
			}
			bodies = append(bodies, tmpBody)
		}
		fmt.Println("show header")
		fmt.Println(headers)
		fmt.Println("show body")
		fmt.Println(bodies)

		// filename: 0-indexed -> 1-indexed
		// ex. 20240501_1.csv
		fileDir := filepath.Join(s.common.outputDir, fmt.Sprintf("%s_%d.csv", s.yyyymmdd, i+1))
		if err := csv.WriteFile(fileDir, headers, bodies); err != nil {
			return err
		}
		slog.Info("write csv complete", "outputFile", fileDir)
	}

	return nil
}

func (s *ScenarioSBI) Start(ctx context.Context) error {
	slog.Info("connect to browser")
	if err := s.getBrowser(ctx); err != nil {
		slog.Error("get browser error", "err", err.Error())
		return err
	}
	defer s.browser.Close()

	if err := s.login(ctx); err != nil {
		return err
	}

	if err := s.getPortfolio(ctx); err != nil {
		return err
	}

	return nil
}
