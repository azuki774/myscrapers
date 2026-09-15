package sbi

import (
	"strings"
	"testing"
)

func TestTickerFromUSToken(t *testing.T) {
	tests := map[string]string{
		"AMDNASDAQ":       "AMD",
		"BRK-BNYSE":       "BRK-B",
		"BRK.BNYSE":       "BRK.B",
		"SPYNYSEArca":     "SPY",
		"ABCNYSEAmerican": "ABC",
		"XYZCboe":         "XYZ",
		"ABCDOTC":         "ABCD",
		"UNKNOWN":         "",
	}
	for input, want := range tests {
		if got := tickerFromUSToken(input); got != want {
			t.Errorf("tickerFromUSToken(%q) = %q, want %q", input, got, want)
		}
	}
}

func TestParseFundRowsStrictMatchesNormalizedHTMLName(t *testing.T) {
	tokens := strings.Fields("積立 売却 ABC ファンド --/--/-- 1 2 3 4 5 6 7 8 詳細")
	holdings, err := parseFundRowsStrict(tokens, `<table><tr><td>ＡＢＣ　ファンド</td><td><a href="/x?fund_sec_code=1234ABCD">詳細</a></td></tr></table>`)
	if err != nil {
		t.Fatalf("parseFundRowsStrict: %v", err)
	}
	if got := holdings[0].source; got == nil || *got != (FIGILookup{Ticker: "1234ABCD", ExchCode: "JP"}) {
		t.Fatalf("source = %+v", got)
	}
}

func TestParseFundRowsStrictRejectsAmbiguousAndMissing(t *testing.T) {
	tokens := strings.Fields("積立 売却 ABCファンド --/--/-- 1 2 3 4 5 6 7 8 詳細")
	tests := []string{
		`<table><tr><td>ABCファンド</td><td><a href="?fund_sec_code=1234ABCD">詳細</a></td></tr><tr><td>ABCファンド</td><td><a href="?fund_sec_code=5678EFGH">詳細</a></td></tr></table>`,
		`<table><tr><td>別ファンド</td><td><a href="?fund_sec_code=1234ABCD">詳細</a></td></tr></table>`,
	}
	for _, page := range tests {
		if _, err := parseFundRowsStrict(tokens, page); err == nil {
			t.Errorf("parseFundRowsStrict(%q) succeeded, want error", page)
		}
	}
}

func TestParseFundRowsStrictNestedETGateRows(t *testing.T) {
	tokens := strings.Fields("積立 売却 ABC ファンド --/--/-- 1 2 3 4 5 6 7 8 詳細 積立 売却 XYZ ファンド --/--/-- 1 2 3 4 5 6 7 8 詳細")
	page := `<table><tr><td><table>
 <tr><td><a href="/ETGate/?_ActionID=NoActionID&amp;path=fund%2Fdetail%2F1234ABCD">ＡＢＣ　ファンド</a></td></tr>
 <tr><td><a href="/ETGate/?path=fund%2Fdetail%2F5678EFGH">ＸＹＺ　ファンド</a></td></tr>
 </table></td></tr></table>`
	holdings, err := parseFundRowsStrict(tokens, page)
	if err != nil {
		t.Fatal(err)
	}
	for i, want := range []string{"1234ABCD", "5678EFGH"} {
		if got := holdings[i].source; got == nil || got.Ticker != want {
			t.Fatalf("holding %d source = %+v, want %s", i, got, want)
		}
	}
}

func TestParseFundRowsStrictRejectsInvalidETGatePaths(t *testing.T) {
	tokens := strings.Fields("積立 売却 ABCファンド --/--/-- 1 2 3 4 5 6 7 8 詳細")
	for _, path := range []string{"1234ABCD", "stock/detail/1234ABCD", "fund/detail/1234", "fund/detail/1234ABCD/extra"} {
		page := `<table><tr><td><a href="/ETGate/?path=` + path + `">ABCファンド</a></td></tr></table>`
		if _, err := parseFundRowsStrict(tokens, page); err == nil {
			t.Errorf("accepted invalid path %q", path)
		}
	}
}

func TestParseUSHoldingsStrictSplitMarket(t *testing.T) {
	for _, symbol := range []string{"XLRENYSE Arca", "XLRE NYSE Arca", "XLRENYSEArca"} {
		text := "取引 サンプル ETF " + symbol + " 10 USD 1500 円 2 (0) 8 USD 1200 円 16 USD 2400 円 20 USD 3000 円 +4 USD +600 円 現買 現売 積立"
		holdings, err := parseUSHoldingsStrict(text)
		if err != nil {
			t.Fatalf("%s: %v", symbol, err)
		}
		if len(holdings) != 1 || holdings[0].source.Ticker != "XLRE" || holdings[0].Quantity != 2 || holdings[0].UnitPrice != 10 || holdings[0].ValueJPY != 3000 {
			t.Fatalf("%s: unexpected holding: %+v", symbol, holdings)
		}
	}
}
