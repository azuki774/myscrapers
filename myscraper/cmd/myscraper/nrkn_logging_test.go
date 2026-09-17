package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"testing"

	"github.com/azuki774/myscrapers/myscraper/internal/cli"
	"github.com/azuki774/myscrapers/myscraper/internal/nrkn"
)

func TestNRKNRunnerWritesAssetsAfterLogoutFailureAndSkipsRunCompletion(t *testing.T) {
	const fixture = `<div class="infoHdWrap"><dl><dt>商品名</dt><dd>DCニッセイ国内株式インデックス</dd><dt>商品コード</dt><dd>29316149</dd><dt>商品分類</dt><dd>国内投信</dd></dl></div><table><tr><th>数量（残高）</th><th>基準価額</th><th>資産評価額</th><th>取得価額累計</th></tr><tr><td>1</td><td>100円</td><td>100円</td><td>90円</td></tr><tr><th>解約価額</th><th>解約時評価額</th><th>損益</th></tr><tr><td>100円</td><td>100円</td><td>10円</td></tr><tr><th>基準日</th><th>資産比率</th></tr><tr><td>2026/09/11</td><td>100％</td></tr></table><table><tr><th>資産評価額合計</th><th>取得価額累計合計</th><th>損益合計</th></tr><tr><td>100円</td><td>90円</td><td>10円</td></tr></table>`
	oldSessionFactory := newNRKNSession
	oldResolverFactory := newNRKNFIGIResolver
	defer func() {
		newNRKNSession = oldSessionFactory
		newNRKNFIGIResolver = oldResolverFactory
	}()
	newNRKNSession = func(context.Context, bool) (nrkn.Session, error) {
		return &logoutFailSession{body: fixture, closeErr: errors.New("browser close unavailable")}, nil
	}
	newNRKNFIGIResolver = func() nrkn.HoldingFIGIResolver { return testNRKNResolver{} }
	t.Setenv("NRKN_ID", "id")
	t.Setenv("NRKN_PASS", "pass")
	t.Setenv("NRKN_BIRTHDAY", "20000101")

	var stdout, stderr bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&stderr, nil))
	code := cli.RunNRKN([]string{"nrkn"}, &stdout, &stderr, logger, nrknRunner{logger: logger, stdout: &stdout})
	if code != 1 {
		t.Fatalf("exit = %d, want 1; logs=%s", code, stderr.String())
	}
	var assets nrkn.Assets
	if err := json.Unmarshal(stdout.Bytes(), &assets); err != nil {
		t.Fatalf("stdout is not assets JSON: %v; stdout=%q", err, stdout.String())
	}
	if assets.GrandTotalJPY != 100 {
		t.Fatalf("grand total = %d, want 100", assets.GrandTotalJPY)
	}

	var records []map[string]any
	decoder := json.NewDecoder(&stderr)
	for {
		var record map[string]any
		err := decoder.Decode(&record)
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatalf("decode log: %v", err)
		}
		records = append(records, record)
	}
	for _, record := range records {
		if record["stage"] == "run" && record["event"] == "completed" {
			t.Fatal("run completed despite logout failure")
		}
	}
	if !hasOperationRecord(records, "cleanup", "warning", "logout") {
		t.Fatal("logout warning record missing")
	}
	if hasOperationRecord(records, "cleanup", "completed", "logout") {
		t.Fatal("logout cleanup completed record emitted after failure")
	}
	if !hasOperationRecord(records, "cleanup", "warning", "browser_close") {
		t.Fatal("browser close warning record missing")
	}
	if hasOperationRecord(records, "cleanup", "completed", "browser_close") {
		t.Fatal("browser close completed record emitted after failure")
	}
}

func hasOperationRecord(records []map[string]any, stage, event, operation string) bool {
	for _, record := range records {
		if record["stage"] == stage && record["event"] == event && record["operation"] == operation {
			return true
		}
	}
	return false
}

type logoutFailSession struct {
	body     string
	closeErr error
}

func (*logoutFailSession) Login(context.Context, string, string, string) error { return nil }
func (*logoutFailSession) NavigateToAssets(context.Context) error              { return nil }
func (s *logoutFailSession) BodyHTML(context.Context) (string, error)          { return s.body, nil }
func (*logoutFailSession) Logout(context.Context) error                        { return errors.New("logout unavailable") }
func (s *logoutFailSession) Close() error                                      { return s.closeErr }

type testNRKNResolver struct{}

func (testNRKNResolver) Resolve(context.Context, []nrkn.Holding) (map[string]string, error) {
	return map[string]string{"29316149": "BBGTESTFUND"}, nil
}
