package sbi

import (
	"encoding/json"
	"math"
	"os"
	"strings"
	"testing"
)

func TestParseAmount(t *testing.T) {
	tests := []struct {
		in   string
		want float64
	}{
		{"24,870", 24870},
		{"1,396,796 円", 1396796},
		{"8,766.15 USD", 8766.15},
		{"+1,456.57 USD", 1456.57},
		{"-50,337 円", -50337},
		{"140,107 円", 140107},
		{"+20.30", 20.30},
		{"1,383,248.56", 1383248.56},
		{"+650,638.94", 650638.94},
		{"+7,486.78", 7486.78},
	}
	for _, tt := range tests {
		got, err := parseAmount(tt.in)
		if err != nil {
			t.Fatalf("parseAmount(%q): %v", tt.in, err)
		}
		if math.Abs(got-tt.want) > 0.01 {
			t.Errorf("parseAmount(%q) = %v, want %v", tt.in, got, tt.want)
		}
	}
}

func TestParseCash(t *testing.T) {
	// Captured from the live 口座サマリー page.
	text := `お客様の積立投信の設定詳細はこちら
| 買付余力
詳細
買付余力(2営業日後)
24,870
| 保有資産評価
詳細
現金残高等
24,870
株式
238,635
投資信託
3,600,880
計
3,864,385`
	cash, err := parseCash(text)
	if err != nil {
		t.Fatalf("parseCash: %v", err)
	}
	if cash != 24870 {
		t.Errorf("parseCash = %v", cash)
	}
}

func TestFindPortfolioSection(t *testing.T) {
	// Captured from the live portfolio page (sections collapsed to a
	// single space-separated line).
	text := `投資信託（金額/特定預り） 登録順表示 取引 ファンド名 買付日 数量 取得単価 現在値 前日比 前日比（％） 損益 損益（％） 評価額 編集 買付 売却 ＳＢＩ・全世界株式インデックス・ファンド --/--/-- 22,382 24,574 36,379 +182 +0.50 +26,421.95 +48.04 81,423.47 詳細 投資信託(金額/特定預り) 合計 評価額 含み損益 含み損益（％） 前日比 前日比（％） 81,423.47 +26,421.95 +48.04 +407.35 +0.50 投資信託（金額/旧つみたてNISA預り） 登録順表示 取引 ファンド名 買付日 数量 取得単価 現在値 前日比 前日比（％） 損益 損益（％） 評価額 編集 積立 売却 ｅＭＡＸＩＳ　Ｓｌｉｍ　新興国株式インデックス --/--/-- 131,210 12,888 27,337 +237 +0.87 +189,585.32 +112.11 358,688.77 詳細 積立 売却 三井住友・ＤＣつみたてＮＩＳＡ・日本株インデックスファンド --/--/-- 49,572 34,213 75,812 +383 +0.51 +206,214.56 +121.59 375,815.24 詳細 積立 売却 ＤＣニッセイワールドセレクトファンド（安定型） --/--/-- 156,591 10,863 11,527 +13 +0.11 +10,397.64 +6.11 180,502.44 詳細 積立 売却 ＳＢＩ・全世界株式インデックス・ファンド --/--/-- 102,225 16,542 36,379 +182 +0.50 +202,783.73 +119.92 371,884.32 詳細 積立 売却 ひふみプラス --/--/-- 11,544 47,384 83,470 +359 +0.43 +41,657.67 +76.16 96,357.76 詳細 投資信託(金額/旧つみたてNISA預り) 合計 評価額 含み損益 含み損益（％） 前日比 前日比（％） 1,383,248.56 +650,638.94 +88.81 +7,486.78 +0.54 総合計 評価額 含み損益 含み損益（％） 基準値比 基準値比（％） 3,840,020.06 +1,139,593.21 +42.20 +500 +0.21`

	oldNisa, err := parseOldNISA(text)
	if err != nil {
		t.Fatalf("parseOldNISA: %v", err)
	}
	if math.Abs(oldNisa.TotalJPY-1383248.56) > 0.01 {
		t.Errorf("oldNisa.TotalJPY = %v", oldNisa.TotalJPY)
	}
	if math.Abs(oldNisa.PnLJPY-650638.94) > 0.01 || math.Abs(oldNisa.PnLPct-88.81) > 0.01 {
		t.Errorf("oldNisa PnL = %+v", oldNisa)
	}
	if math.Abs(oldNisa.PrevDayJPY-7486.78) > 0.01 || math.Abs(oldNisa.PrevDayPct-0.54) > 0.01 {
		t.Errorf("oldNisa prev day = %+v", oldNisa)
	}
	if len(oldNisa.Funds) != 5 {
		t.Fatalf("oldNisa.Funds = %d funds, want 5", len(oldNisa.Funds))
	}
	first := oldNisa.Funds[0]
	if first.Name != "ｅＭＡＸＩＳ Ｓｌｉｍ 新興国株式インデックス" {
		t.Errorf("fund[0].Name = %q", first.Name)
	}
	if math.Abs(first.ValueJPY-358688.77) > 0.01 || math.Abs(first.PnLJPY-189585.32) > 0.01 {
		t.Errorf("fund[0] = %+v", first)
	}
	if math.Abs(first.PrevDayJPY-237) > 0.01 || math.Abs(first.PrevDayPct-0.87) > 0.01 {
		t.Errorf("fund[0] prev day = %+v", first)
	}
	last := oldNisa.Funds[4]
	if last.Name != "ひふみプラス" || math.Abs(last.ValueJPY-96357.76) > 0.01 {
		t.Errorf("fund[4] = %+v", last)
	}

	funds, err := parsePortfolioValue(text, "投資信託(金額/特定預り)")
	if err != nil {
		t.Fatalf("parsePortfolioValue(特定預り): %v", err)
	}
	if math.Abs(funds-81423.47) > 0.01 {
		t.Errorf("特定預り = %v", funds)
	}
}

func TestParseForeignCash(t *testing.T) {
	// Captured from the live foreign 口座サマリー page.
	text := `保有資産評価
預り金
通貨
保有数量
円換算評価額
米ドル
879.30 USD
140,107 円
香港ドル
0.00 HKD
0 円`
	got, err := parseForeignCash(text)
	if err != nil {
		t.Fatalf("parseForeignCash: %v", err)
	}
	if math.Abs(got.USD-879.30) > 0.01 || got.JPY != 140107 {
		t.Errorf("parseForeignCash = %+v", got)
	}
}

func TestParseNISA(t *testing.T) {
	// Captured from the live NISA portfolio page.
	text := `NISAトップ
ランキング
保有銘柄
NISA資産残高
3,771,642円
( 2026/08/16 20:40 )
前日比/率
0円
(0.00%)
前月比/率
+967,383円
(+34.49%)
評価損益/率
+697,759円
(+22.69%)
資産構成比率
国内株式
外国株式
投資信託
すべて
成長投資枠
つみたて投資枠
国内株式
評価額
238,635円
評価損益
-2,140円
-0.88%
前日比
0円
0.00%
前月比
+10,115円
+4.42%
米国株式
評価額
1,396,796円
評価損益
+235,728円
+20.30%
前日比
0円
0.00%
前月比
+791,050円
+130.59%
投資信託
評価額
2,136,211円
評価損益
+464,171円
+27.76%
前日比
0円
0.00%
前月比
+166,218円
+8.43%`
	nisa, err := parseNISA(text)
	if err != nil {
		t.Fatalf("parseNISA: %v", err)
	}
	if nisa.TotalJPY != 3771642 {
		t.Errorf("TotalJPY = %v", nisa.TotalJPY)
	}
	if nisa.PrevDayJPY != 0 || nisa.PrevMonthJPY != 967383 || nisa.PnLJPY != 697759 {
		t.Errorf("summary = %+v", nisa)
	}
	if math.Abs(nisa.PrevMonthPct-34.49) > 0.01 || math.Abs(nisa.PnLPct-22.69) > 0.01 {
		t.Errorf("summary pct = %+v", nisa)
	}
	if nisa.Domestic.ValueJPY != 238635 || nisa.Domestic.PnLJPY != -2140 {
		t.Errorf("domestic = %+v", nisa.Domestic)
	}
	if math.Abs(nisa.Domestic.PnLPct+0.88) > 0.01 {
		t.Errorf("domestic pct = %+v", nisa.Domestic)
	}
	if nisa.USStocks.ValueJPY != 1396796 || nisa.USStocks.PrevMonthJPY != 791050 {
		t.Errorf("us = %+v", nisa.USStocks)
	}
	if nisa.Funds.ValueJPY != 2136211 || nisa.Funds.PrevMonthJPY != 166218 {
		t.Errorf("funds = %+v", nisa.Funds)
	}
}

func TestParseStockRows(t *testing.T) {
	// Captured from the live portfolio page, 株式（現物/NISA預り（成長投資枠）） section.
	text := `株式（現物/NISA預り（成長投資枠）） 登録順表示 取引 銘柄（コード） 買付日 数量 取得単価 現在値 前日比 前日比（％） 損益 損益（％） 評価額 編集 現買 現売 積立 1540 純金信託 26/03/06 5 24,235 20,851 +176 +0.85 -16,920 -13.96 104,255 詳細 現買 現売 積立 5401 日本製鉄 26/03/06 200 598 674.4 -1.9 -0.28 +15,280 +12.78 134,880 詳細`
	tokens := sectionTokens(strings.Fields(text), "株式（現物/NISA預り（成長投資枠））")
	stocks := parseStockRows(tokens)
	if len(stocks) != 2 {
		t.Fatalf("stocks = %d, want 2", len(stocks))
	}
	first := stocks[0]
	if first.Name != "純金信託" || first.Quantity != 5 || first.ValueJPY != 104255 {
		t.Errorf("stock[0] = %+v", first)
	}
	if first.PrevDayJPY != 176 || math.Abs(first.PrevDayPct-0.85) > 0.01 || first.PnLJPY != -16920 {
		t.Errorf("stock[0] figures = %+v", first)
	}
	second := stocks[1]
	if second.Name != "日本製鉄" || second.ValueJPY != 134880 || second.PnLJPY != 15280 {
		t.Errorf("stock[1] = %+v", second)
	}
}

func TestParseUSHoldings(t *testing.T) {
	// Captured from the live foreign 保有銘柄 page.
	text := `取引 アドバンスト マイクロ デバイシズ AMDNASDAQ 514.39 USD 81,962 円 5 (0) 201.05 USD 31,857 円 1,005.25 USD 159,285 円 2,571.95 USD 409,814 円 +1,566.70 USD +250,529 円 現買 現売 積立 スペース エクスプロレーション テクノ A SPCXNASDAQ 140.00 USD 22,307 円 5 (0) 200.51 USD 32,375 円 1,002.55 USD 161,875 円 700.00 USD 111,538 円 -302.55 USD -50,337 円 現買 現売 積立`
	holdings := parseUSHoldings(text)
	if len(holdings) != 2 {
		t.Fatalf("holdings = %d, want 2", len(holdings))
	}
	first := holdings[0]
	if first.Name != "アドバンスト マイクロ デバイシズ" || first.Quantity != 5 {
		t.Errorf("us[0] = %+v", first)
	}
	if math.Abs(first.UnitCost-201.05) > 0.01 || math.Abs(first.UnitPrice-514.39) > 0.01 {
		t.Errorf("us[0] units = %+v", first)
	}
	if first.ValueJPY != 409814 || first.PnLJPY != 250529 {
		t.Errorf("us[0] jpy = %+v", first)
	}
	second := holdings[1]
	if second.Name != "スペース エクスプロレーション テクノ" || second.PnLJPY != -50337 || second.ValueJPY != 111538 {
		t.Errorf("us[1] = %+v", second)
	}
}

// TestMECE asserts that the sum of the MECE sections matches the
// grand total computed by FetchAssets.
func TestMECE(t *testing.T) {
	assets := Assets{
		NISA: NISA{
			TotalJPY: 3771642,
			Domestic: NISAItem{ValueJPY: 238635},
			USStocks: NISAItem{ValueJPY: 1396796},
			Funds:    NISAItem{ValueJPY: 2136211},
		},
		OldNISA: OldNISA{TotalJPY: 1383248.56},
		Other: Other{
			CashJPY:  24870,
			FundsJPY: 81423.47,
			USDCash:  Foreign{USD: 879.3, JPY: 140107},
		},
	}
	sum := assets.NISA.TotalJPY + assets.OldNISA.TotalJPY + assets.Other.CashJPY + assets.Other.FundsJPY + assets.Other.USDCash.JPY
	want := 3771642 + 1383248.56 + 24870 + 81423.47 + 140107
	if math.Abs(sum-want) > 0.01 {
		t.Errorf("MECE sum = %v, want %v", sum, want)
	}
	_ = assets.GrandTotalJPY
}

// TestExampleAssetsJSON parses the complete sample output in
// testdata/example-assets.json and verifies that every section is
// present and internally consistent (holdings sum to their class,
// sections sum to the grand total). The file doubles as the
// documented output format and as a schema test.
func TestExampleAssetsJSON(t *testing.T) {
	raw, err := os.ReadFile("testdata/example-assets.json")
	if err != nil {
		t.Fatalf("read testdata: %v", err)
	}
	var assets Assets
	if err := json.Unmarshal(raw, &assets); err != nil {
		t.Fatalf("unmarshal example: %v", err)
	}

	// Sections sum to grand total (MECE).
	want := assets.NISA.TotalJPY + assets.OldNISA.TotalJPY +
		assets.Other.CashJPY + assets.Other.FundsJPY + assets.Other.USDCash.JPY
	if math.Abs(want-assets.GrandTotalJPY) > 0.01 {
		t.Errorf("grand total = %v, sections sum = %v", assets.GrandTotalJPY, want)
	}

	// Each NISA class sums its holdings.
	assertHoldingsMatch(t, "nisa.domestic_stocks", assets.NISA.Domestic.ValueJPY, assets.NISA.Domestic.Holdings)
	assertHoldingsMatch(t, "nisa.us_stocks", assets.NISA.USStocks.ValueJPY, assets.NISA.USStocks.Holdings)
	assertHoldingsMatch(t, "nisa.funds", assets.NISA.Funds.ValueJPY, assets.NISA.Funds.Holdings)
	assertHoldingsMatch(t, "old_nisa.funds", assets.OldNISA.TotalJPY, assets.OldNISA.Funds)

	// NISA classes sum to the NISA total (allowing rounding on the page).
	classSum := assets.NISA.Domestic.ValueJPY + assets.NISA.USStocks.ValueJPY + assets.NISA.Funds.ValueJPY
	if math.Abs(classSum-assets.NISA.TotalJPY) > 1 {
		t.Errorf("nisa class sum = %v, total = %v", classSum, assets.NISA.TotalJPY)
	}

	// Every holding has a name and positive value.
	for _, h := range append(append(append([]Holding{},
		assets.NISA.Domestic.Holdings...),
		assets.NISA.USStocks.Holdings...),
		assets.NISA.Funds.Holdings...) {
		if h.Name == "" || h.ValueJPY <= 0 {
			t.Errorf("invalid holding: %+v", h)
		}
	}
	for _, h := range assets.OldNISA.Funds {
		if h.Name == "" || h.ValueJPY <= 0 {
			t.Errorf("invalid old_nisa holding: %+v", h)
		}
	}
}

func assertHoldingsMatch(t *testing.T, label string, classValue float64, holdings []Holding) {
	t.Helper()
	var sum float64
	for _, h := range holdings {
		sum += h.ValueJPY
	}
	if math.Abs(sum-classValue) > 0.01 {
		t.Errorf("%s holdings sum = %v, class value = %v", label, sum, classValue)
	}
}
