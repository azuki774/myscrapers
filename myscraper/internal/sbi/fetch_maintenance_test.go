package sbi

import (
	"context"
	"testing"
	"time"
)

// Body text fixtures for FetchAssets orchestration. Values are
// synthetic round numbers, not real account data; each fixture only
// needs to satisfy the structure that the corresponding parser expects.
const (
	cashFixture = `保有資産評価
現金残高等
12,345
株式
6,789
投資信託
1,234
計
20,368`

	portfolioFixture = `投資信託（金額/特定預り） 登録順表示 取引 ファンド名 買付日 数量 取得単価 現在値 前日比 前日比（％） 損益 損益（％） 評価額 編集 買付 売却 サンプルファンド --/--/-- 1,000 1,000 1,000 +1 +0.10 +100 +10.0 1,000 詳細 投資信託(金額/特定預り) 合計 評価額 含み損益 含み損益（％） 前日比 前日比（％） 2,000 +200 +10.0 +10 +0.50 投資信託（金額/旧つみたてNISA預り） 登録順表示 取引 ファンド名 買付日 数量 取得単価 現在値 前日比 前日比（％） 損益 損益（％） 評価額 編集 積立 売却 サンプル投信 --/--/-- 2,000 500 1,000 +2 +0.20 +200 +20.0 2,000 詳細 投資信託(金額/旧つみたてNISA預り) 合計 評価額 含み損益 含み損益（％） 前日比 前日比（％） 4,000 +400 +10.0 +20 +0.50 総合計 評価額 含み損益 含み損益（％） 基準値比 基準値比（％） 6,000 +600 +10.0 +30 +0.50`

	nisaFixture = `NISAトップ
ランキング
保有銘柄
NISA資産残高
10,000円
( 2026/08/16 20:40 )
前日比/率
100円
(+1.00%)
前月比/率
200円
(+2.00%)
評価損益/率
300円
(+3.00%)
資産構成比率
国内株式
外国株式
投資信託
すべて
成長投資枠
つみたて投資枠
国内株式
評価額
1,000円
評価損益
10円
1.00%
前日比
1円
0.10%
前月比
2円
0.20%
米国株式
評価額
2,000円
評価損益
20円
1.00%
前日比
2円
0.10%
前月比
4円
0.20%
投資信託
評価額
3,000円
評価損益
30円
1.00%
前日比
3円
0.10%
前月比
6円
0.20%`

	foreignHoldingsFixture = `サンプル米国株 SAMPLEX 100.00 USD 10,000 円 5 (0) 90.00 USD 9,000 円 450.00 USD 45,000 円 500.00 USD 50,000 円 +50.00 USD +5,000 円 現買 現売 積立`

	foreignCashFixture = `保有資産評価
預り金
通貨
保有数量
円換算評価額
米ドル
100.00 USD
15,000 円
香港ドル
0.00 HKD
0 円`

	maintenanceFixture = `臨時メンテナンスのお知らせ

日頃よりSBI証券をご利用いただきありがとうございます。

当サイトは臨時メンテナンスを実施中のため、一時的にサービスのご利用ができません。
しばらくお待ちください。`
)

func TestIsMaintenancePage(t *testing.T) {
	tests := []struct {
		name string
		text string
		want bool
	}{
		{
			name: "temporary maintenance notice",
			text: `臨時メンテナンスのお知らせ

日頃よりSBI証券をご利用いただきありがとうございます。

当サイトは臨時メンテナンスを実施中のため、一時的にサービスのご利用ができません。
しばらくお待ちください。`,
			want: true,
		},
		{
			name: "scheduled maintenance notice",
			text: `メンテナンスのお知らせ
NISAサービスはメンテナンスのため、サービスのご利用ができません。`,
			want: true,
		},
		{
			name: "normal nisa page",
			text: `NISA資産残高
3,771,642円
前日比/率
0円
(0.00%)`,
			want: false,
		},
		{
			name: "normal portfolio page",
			text: `投資信託（金額/特定預り）合計
81,423.47`,
			want: false,
		},
		{
			name: "empty page",
			text: ``,
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isMaintenancePage(tt.text); got != tt.want {
				t.Errorf("isMaintenancePage() = %v, want %v", got, tt.want)
			}
		})
	}
}

// fakeSession implements Session with a canned body per URL so the
// FetchAssets orchestration can be exercised without a browser.
type fakeSession struct {
	bodies map[string]string
	last   string
}

func (f *fakeSession) LoginWithPasskey(context.Context, *PasskeyFile) error { return nil }
func (f *fakeSession) Goto(_ context.Context, url string) error {
	f.last = url
	return nil
}
func (f *fakeSession) BodyText(context.Context) (string, error) {
	return f.bodies[f.last], nil
}
func (f *fakeSession) Wait(context.Context, time.Duration) error { return nil }
func (f *fakeSession) Close() error                              { return nil }

// TestFetchAssetsMaintenanceStatus verifies that when the NISA page is
// an SBI maintenance page, FetchAssets returns an Assets with
// Status=maintenance instead of failing, while keeping the sections
// that were still parseable (portfolio-derived sections).
func TestFetchAssetsMaintenanceStatus(t *testing.T) {
	sess := &fakeSession{bodies: map[string]string{
		portfolioURL:       portfolioFixture,
		nisaPortfolioURL:   maintenanceFixture,
		foreignAssetsURL:   foreignHoldingsFixture,
		domesticSummaryURL: cashFixture,
		foreignSummaryURL:  foreignCashFixture,
	}}
	assets, err := FetchAssets(context.Background(), sess, time.Now())
	if err != nil {
		t.Fatalf("FetchAssets: %v", err)
	}
	if assets.Status != StatusMaintenance {
		t.Errorf("Status = %q, want %q", assets.Status, StatusMaintenance)
	}
	if assets.NISA.TotalJPY != 0 {
		t.Errorf("NISA.TotalJPY = %v, want 0 on maintenance", assets.NISA.TotalJPY)
	}
	if assets.OldNISA.TotalJPY != 4000 {
		t.Errorf("OldNISA.TotalJPY = %v, want collected portfolio data", assets.OldNISA.TotalJPY)
	}
	if assets.Cash.JPY.ValueJPY != 12345 {
		t.Errorf("Cash.JPY.ValueJPY = %v, want collected cash data", assets.Cash.JPY.ValueJPY)
	}
}

// TestFetchAssetsOKStatus verifies that a full healthy fetch reports
// Status=ok.
func TestFetchAssetsOKStatus(t *testing.T) {
	sess := &fakeSession{bodies: map[string]string{
		portfolioURL:       portfolioFixture,
		nisaPortfolioURL:   nisaFixture,
		foreignAssetsURL:   foreignHoldingsFixture,
		domesticSummaryURL: cashFixture,
		foreignSummaryURL:  foreignCashFixture,
	}}
	assets, err := FetchAssets(context.Background(), sess, time.Now())
	if err != nil {
		t.Fatalf("FetchAssets: %v", err)
	}
	if assets.Status != StatusOK {
		t.Errorf("Status = %q, want %q", assets.Status, StatusOK)
	}
	if assets.NISA.TotalJPY != 10000 {
		t.Errorf("NISA.TotalJPY = %v", assets.NISA.TotalJPY)
	}
}
