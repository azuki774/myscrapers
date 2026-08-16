package sbi

import (
	"context"
	"fmt"
	"log/slog"
	"regexp"
	"strconv"
	"strings"
	"time"
	"unicode"
)

// FetchOptions carries the inputs for a full SBI asset scrape.
type FetchOptions struct {
	PasskeyPath string
	Passkey     *PasskeyFile
	OutputPath  string
	Now         time.Time
	Logger      *slog.Logger
	Headless    bool
}

// Fixed page URLs. These never change for a given login session; the
// scraper hard-codes them instead of navigating via menus.
const (
	domesticSummaryURL = "https://www.sbisec.co.jp/ETGate/?_ControlID=WPLETacR001Control&_PageID=DefaultPID&_ActionID=DefaultAID&_DataStoreID=DSWPLETacR001Control&OutSide=on&getFlg=on"
	portfolioURL       = "https://site1.sbisec.co.jp/ETGate/?_ControlID=WPLETpfR001Control&_PageID=DefaultPID&_ActionID=DefaultAID&_DataStoreID=DSWPLETpfR001Control&OutSide=on&getFlg=on&_scpr=intpr=hn_trade"
	foreignAssetsURL   = "https://member.c.sbisec.co.jp/foreign/account/assets"
	foreignSummaryURL  = "https://member.c.sbisec.co.jp/foreign/account/summary"
	nisaPortfolioURL   = "https://site.sbisec.co.jp/account/nisa/portfolio"
)

// Assets is the consolidated, MECE result of a full scrape. Every JPY
// figure is accounted for exactly once across nisa / old_nisa / other,
// so grand_total_jpy is the plain sum of the sections.
type Assets struct {
	FetchedAt     time.Time `json:"fetched_at"`
	NISA          NISA      `json:"nisa"`
	OldNISA       OldNISA   `json:"old_nisa"`
	Other         Other     `json:"other"`
	GrandTotalJPY float64   `json:"grand_total_jpy"`
}

// Foreign holds USD-denominated amounts plus their JPY conversion.
type Foreign struct {
	USD float64 `json:"usd"`
	JPY float64 `json:"jpy"`
}

// NISA is the new-NISA portfolio (国内株式 + 米国株式 + 投資信託) with
// day-over-day, month-over-month, and cumulative P&L plus a
// per-asset-class breakdown. Data source: NISA portfolio page.
type NISA struct {
	TotalJPY     float64  `json:"total_jpy"`
	PrevDayJPY   float64  `json:"prev_day_jpy"`
	PrevDayPct   float64  `json:"prev_day_pct"`
	PrevMonthJPY float64  `json:"prev_month_jpy"`
	PrevMonthPct float64  `json:"prev_month_pct"`
	PnLJPY       float64  `json:"pnl_jpy"`
	PnLPct       float64  `json:"pnl_pct"`
	Domestic     NISAItem `json:"domestic_stocks"`
	USStocks     NISAItem `json:"us_stocks"`
	Funds        NISAItem `json:"funds"`
}

// NISAItem is the per-asset-class breakdown on the NISA portfolio page,
// with the per-holding breakdown from the portfolio / foreign pages.
type NISAItem struct {
	ValueJPY     float64   `json:"value_jpy"`
	PnLJPY       float64   `json:"pnl_jpy"`
	PnLPct       float64   `json:"pnl_pct"`
	PrevDayJPY   float64   `json:"prev_day_jpy"`
	PrevDayPct   float64   `json:"prev_day_pct"`
	PrevMonthJPY float64   `json:"prev_month_jpy"`
	PrevMonthPct float64   `json:"prev_month_pct"`
	Holdings     []Holding `json:"holdings"`
}

// Holding is a single security position. Unit price/cost and the
// prev-day figures are per unit (or per 10k units for funds); for US
// stocks the per-unit figures are in USD while P&L and value are in JPY
// (prev-day figures are unavailable for US stocks).
type Holding struct {
	Name       string  `json:"name"`
	Quantity   float64 `json:"quantity"`
	UnitCost   float64 `json:"unit_cost"`
	UnitPrice  float64 `json:"unit_price"`
	PrevDayJPY float64 `json:"prev_day_jpy"`
	PrevDayPct float64 `json:"prev_day_pct"`
	PnLJPY     float64 `json:"pnl_jpy"`
	PnLPct     float64 `json:"pnl_pct"`
	ValueJPY   float64 `json:"value_jpy"`
}

// OldNISA is the 旧つみたてNISA 投資信託 holdings. SBI offers no
// month-over-month figure for this section, so only value, prev-day
// change, and cumulative P&L are present, plus a per-fund breakdown.
// Data source: portfolio page.
type OldNISA struct {
	TotalJPY   float64   `json:"total_jpy"`
	PrevDayJPY float64   `json:"prev_day_jpy"`
	PrevDayPct float64   `json:"prev_day_pct"`
	PnLJPY     float64   `json:"pnl_jpy"`
	PnLPct     float64   `json:"pnl_pct"`
	Funds      []Holding `json:"funds"`
}

// Other holds everything outside NISA: 現金残高, 特定預り 投資信託, and
// the USD deposit. Data sources: account summary and foreign summary.
type Other struct {
	CashJPY  float64 `json:"cash_jpy"`
	FundsJPY float64 `json:"funds_jpy"`
	USDCash  Foreign `json:"usd_cash"`
}

// FetchAssets scrapes the four fixed pages and returns the
// consolidated asset summary. The sections are mutually exclusive and
// collectively exhaustive: nisa (new NISA), old_nisa (旧つみたてNISA),
// and other (cash + 特定預り投信 + USD deposit).
func FetchAssets(ctx context.Context, sess Session, now time.Time) (*Assets, error) {
	// 1. Portfolio page: 旧つみたてNISA and 特定預り 投資信託 sections,
	// plus the NISA 国内株式 / NISA 投信 holding rows.
	if err := sess.Goto(ctx, portfolioURL); err != nil {
		return nil, err
	}
	if err := sess.Wait(ctx, 3*time.Second); err != nil {
		return nil, err
	}
	portfolioText, err := sess.BodyText(ctx)
	if err != nil {
		return nil, err
	}
	oldNisa, err := parseOldNISA(portfolioText)
	if err != nil {
		return nil, fmt.Errorf("parse old nisa: %w", err)
	}
	otherFunds, err := parsePortfolioValue(portfolioText, "投資信託(金額/特定預り)")
	if err != nil {
		return nil, fmt.Errorf("parse 特定預り: %w", err)
	}
	portfolioTokens := strings.Fields(portfolioText)
	domesticHoldings := parseStockRows(sectionTokens(portfolioTokens, "株式（現物/NISA預り（成長投資枠））"))
	fundHoldings := parseFundRows(sectionTokens(portfolioTokens, "投資信託（金額/NISA預り（つみたて投資枠））"))

	// 2. NISA portfolio page (total balance, prev-day/prev-month/P&L).
	if err := sess.Goto(ctx, nisaPortfolioURL); err != nil {
		return nil, err
	}
	if err := sess.Wait(ctx, 3*time.Second); err != nil {
		return nil, err
	}
	nisaText, err := sess.BodyText(ctx)
	if err != nil {
		return nil, err
	}
	nisa, err := parseNISA(nisaText)
	if err != nil {
		return nil, fmt.Errorf("parse nisa portfolio: %w", err)
	}
	nisa.Domestic.Holdings = domesticHoldings
	nisa.Funds.Holdings = fundHoldings

	// 3. Foreign 保有銘柄 page (US stock holding rows for NISA).
	if err := sess.Goto(ctx, foreignAssetsURL); err != nil {
		return nil, err
	}
	if err := sess.Wait(ctx, 3*time.Second); err != nil {
		return nil, err
	}
	assetsText, err := sess.BodyText(ctx)
	if err != nil {
		return nil, err
	}
	nisa.USStocks.Holdings = parseUSHoldings(assetsText)

	// 4. Domestic 口座サマリー page (現金残高等).
	if err := sess.Goto(ctx, domesticSummaryURL); err != nil {
		return nil, err
	}
	if err := sess.Wait(ctx, 3*time.Second); err != nil {
		return nil, err
	}
	domText, err := sess.BodyText(ctx)
	if err != nil {
		return nil, err
	}
	cash, err := parseCash(domText)
	if err != nil {
		return nil, fmt.Errorf("parse cash: %w", err)
	}

	// 5. Foreign 口座サマリー page (USD cash deposit).
	if err := sess.Goto(ctx, foreignSummaryURL); err != nil {
		return nil, err
	}
	if err := sess.Wait(ctx, 3*time.Second); err != nil {
		return nil, err
	}
	summaryText, err := sess.BodyText(ctx)
	if err != nil {
		return nil, err
	}
	usdCash, err := parseForeignCash(summaryText)
	if err != nil {
		return nil, fmt.Errorf("parse foreign cash: %w", err)
	}

	assets := &Assets{
		FetchedAt: now,
		NISA:      nisa,
		OldNISA:   oldNisa,
		Other: Other{
			CashJPY:  cash,
			FundsJPY: otherFunds,
			USDCash:  usdCash,
		},
	}
	assets.GrandTotalJPY = nisa.TotalJPY + oldNisa.TotalJPY + cash + otherFunds + usdCash.JPY
	return assets, nil
}

// amountRe matches the first numeric token in strings like
// "1,396,796 円", "+8,766.15 USD", "-50,337 円".
var amountRe = regexp.MustCompile(`[-+]?[\d,]+(\.\d+)?`)

// parseAmount extracts and parses the numeric value from a page text
// fragment such as "1,396,796 円" or "879.30 USD".
func parseAmount(s string) (float64, error) {
	m := amountRe.FindString(s)
	if m == "" {
		return 0, fmt.Errorf("no amount found in %q", s)
	}
	m = strings.ReplaceAll(m, ",", "")
	v, err := strconv.ParseFloat(m, 64)
	if err != nil {
		return 0, fmt.Errorf("parse amount %q: %w", m, err)
	}
	return v, nil
}

// lines splits body text into trimmed, non-empty lines.
func lines(text string) []string {
	var out []string
	for _, l := range strings.Split(text, "\n") {
		if t := strings.TrimSpace(l); t != "" {
			out = append(out, t)
		}
	}
	return out
}

// valueAfter returns the parsed amount on the line after a line that
// exactly equals key. Exact matching avoids false hits from menu items
// such as 外国株式/国内株式 that merely contain the key as a substring,
// and matches whose next line is not numeric (e.g. nav labels) are
// skipped in favour of later hits in the 保有資産評価 section.
func valueAfter(ls []string, key string) (float64, error) {
	for i, l := range ls {
		if l == key && i+1 < len(ls) {
			if v, err := parseAmount(ls[i+1]); err == nil {
				return v, nil
			}
		}
	}
	return 0, fmt.Errorf("key %q not found", key)
}

// parseCash extracts the 現金残高等 figure from the domestic 口座サマリー
// body text.
func parseCash(text string) (float64, error) {
	return valueAfter(lines(text), "現金残高等")
}

// parseOldNISA extracts the 投資信託(金額/旧つみたてNISA預り) section
// from the portfolio page: section total (value, cumulative P&L,
// prev-day change) plus the per-fund breakdown.
func parseOldNISA(text string) (OldNISA, error) {
	tokens := strings.Fields(text)
	section, err := findPortfolioSection(text, "投資信託(金額/旧つみたてNISA預り)")
	if err != nil {
		return OldNISA{}, err
	}
	oldNisa := OldNISA{
		TotalJPY:   section.value,
		PnLJPY:     section.pnl,
		PnLPct:     section.pnlPct,
		PrevDayJPY: section.prevDay,
		PrevDayPct: section.prevDayPct,
	}
	oldNisa.Funds = parseFundRows(sectionTokens(tokens, "投資信託（金額/旧つみたてNISA預り）"))
	return oldNisa, nil
}

// sectionTokens returns the token slice from the section heading (full
// width parens on the page) up to but not including the section 合計 row.
func sectionTokens(tokens []string, heading string) []string {
	start := -1
	for i, tok := range tokens {
		if tok == heading {
			start = i
			break
		}
	}
	if start < 0 {
		return nil
	}
	end := len(tokens)
	for i := start + 1; i < len(tokens); i++ {
		if tokens[i] == "合計" {
			end = i
			break
		}
	}
	return tokens[start:end]
}

// parseFundRows parses 投資信託 fund rows in the pattern
// 積立 売却 <fund name> --/--/-- 数量 取得単価 現在値 前日比 前日比％
// 損益 損益％ 評価額 詳細.
func parseFundRows(tokens []string) []Holding {
	var funds []Holding
	const valueCount = 8 // 数量 取得単価 現在値 前日比 前日比％ 損益 損益％ 評価額
	for i := 0; i < len(tokens); {
		if tokens[i] != "積立" || i+1 >= len(tokens) || tokens[i+1] != "売却" {
			i++
			continue
		}
		nameEnd := -1
		for j := i + 2; j < len(tokens); j++ {
			if tokens[j] == "--/--/--" {
				nameEnd = j
				break
			}
		}
		if nameEnd < 0 || nameEnd+1+valueCount > len(tokens) {
			i++
			continue
		}
		vals := tokens[nameEnd+1 : nameEnd+1+valueCount]
		parse := func(s string) float64 {
			v, _ := parseAmount(s)
			return v
		}
		funds = append(funds, Holding{
			Name:       strings.Join(tokens[i+2:nameEnd], " "),
			Quantity:   parse(vals[0]),
			UnitCost:   parse(vals[1]),
			UnitPrice:  parse(vals[2]),
			PrevDayJPY: parse(vals[3]),
			PrevDayPct: parse(vals[4]),
			PnLJPY:     parse(vals[5]),
			PnLPct:     parse(vals[6]),
			ValueJPY:   parse(vals[7]),
		})
		i = nameEnd + 1 + valueCount + 1 // skip 詳細
	}
	return funds
}

// parseStockRows parses 株式 rows in the pattern
// 現買 現売 積立 <code> <stock name> <buy date> 数量 取得単価 現在値
// 前日比 前日比％ 損益 損益％ 評価額 詳細.
func parseStockRows(tokens []string) []Holding {
	var stocks []Holding
	const valueCount = 8
	for i := 0; i < len(tokens); {
		if tokens[i] != "現買" || i+1 >= len(tokens) || tokens[i+1] != "現売" || i+2 >= len(tokens) || tokens[i+2] != "積立" {
			i++
			continue
		}
		// After 積立: <code> <name...> <date like 26/03/06> 数量 ...
		if i+3 >= len(tokens) {
			break
		}
		dateIdx := -1
		for j := i + 3; j < len(tokens); j++ {
			if dateRe.MatchString(tokens[j]) {
				dateIdx = j
				break
			}
		}
		if dateIdx < 0 || dateIdx+1+valueCount > len(tokens) {
			i++
			continue
		}
		nameTokens := tokens[i+4 : dateIdx] // skip code
		vals := tokens[dateIdx+1 : dateIdx+1+valueCount]
		parse := func(s string) float64 {
			v, _ := parseAmount(s)
			return v
		}
		stocks = append(stocks, Holding{
			Name:       strings.Join(nameTokens, " "),
			Quantity:   parse(vals[0]),
			UnitCost:   parse(vals[1]),
			UnitPrice:  parse(vals[2]),
			PrevDayJPY: parse(vals[3]),
			PrevDayPct: parse(vals[4]),
			PnLJPY:     parse(vals[5]),
			PnLPct:     parse(vals[6]),
			ValueJPY:   parse(vals[7]),
		})
		i = dateIdx + 1 + valueCount + 1 // skip 詳細
	}
	return stocks
}

var dateRe = regexp.MustCompile(`^\d{2}/\d{2}/\d{2}$`)

// parseUSHoldings parses the foreign 保有銘柄 page rows. Each row
// (from "(0)" back and forth) looks like:
// <name tokens> <ticker+market> <price USD> USD <price JPY> 円 数量 (0)
// <cost USD> USD <cost JPY> 円 <acq amount USD> USD <acq amount JPY> 円
// <value USD> USD <value JPY> 円 <pnl USD> USD <pnl JPY> 円.
// Only JPY value/pnl and USD per-unit figures are exposed; prev-day
// figures are unavailable for US stocks.
func parseUSHoldings(text string) []Holding {
	tokens := strings.Fields(text)
	var holdings []Holding
	for i := 0; i < len(tokens); {
		if tokens[i] != "(0)" {
			i++
			continue
		}
		if i+16 >= len(tokens) || i-5 < 0 {
			break
		}
		parse := func(s string) float64 {
			v, _ := parseAmount(s)
			return v
		}
		rowStart := 0
		for j := i - 1; j >= 0; j-- {
			if tokens[j] == "現買" || tokens[j] == "取引" {
				rowStart = j + 1
				break
			}
		}
		holdings = append(holdings, Holding{
			Name:      strings.Join(nameTokensOnly(tokens[rowStart:i-1]), " "),
			Quantity:  parse(tokens[i-1]),  // 数量
			UnitCost:  parse(tokens[i+1]),  // cost USD
			UnitPrice: parse(tokens[i-5]),  // price USD
			ValueJPY:  parse(tokens[i+11]), // value JPY
			PnLJPY:    parse(tokens[i+15]), // pnl JPY
		})
		i += 17
	}
	return holdings
}

// nameTokensOnly keeps Japanese tokens except currency labels, action
// labels, and numbers, reconstructing holding names from mixed
// name/ticker/running-value token runs.
func nameTokensOnly(tokens []string) []string {
	var name []string
	for _, t := range tokens {
		if !containsJapanese(t) || t == "円" || t == "現買" || t == "現売" || t == "積立" {
			continue
		}
		if _, err := parseAmount(t); err == nil {
			continue
		}
		name = append(name, t)
	}
	return name
}

func containsJapanese(s string) bool {
	for _, r := range s {
		if unicode.Is(unicode.Hiragana, r) || unicode.Is(unicode.Katakana, r) || unicode.Is(unicode.Han, r) {
			return true
		}
	}
	return false
}

// parsePortfolioValue returns the section 評価額 for the given section
// header on the portfolio page.
func parsePortfolioValue(text, key string) (float64, error) {
	section, err := findPortfolioSection(text, key)
	if err != nil {
		return 0, err
	}
	return section.value, nil
}

// portfolioSection is the parsed 合計 row of a portfolio page section.
type portfolioSection struct {
	value     float64
	pnl       float64
	pnlPct    float64
	prevDay   float64
	prevDayPct float64
}

// findPortfolioSection locates the section header token (e.g.
// 投資信託(金額/特定預り)), which is followed by the fixed header
// 合計 評価額 含み損益 含み損益（％） 前日比 前日比（％） and then five
// numeric tokens.
func findPortfolioSection(text, key string) (portfolioSection, error) {
	tokens := strings.Fields(text)
	for i, tok := range tokens {
		if tok != key {
			continue
		}
		if i+1 >= len(tokens) || tokens[i+1] != "合計" {
			continue
		}
		const headerLen = 6 // 合計 評価額 含み損益 含み損益（％） 前日比 前日比（％）
		if i+1+headerLen+5 > len(tokens) {
			continue
		}
		vals := tokens[i+1+headerLen:]
		var sec portfolioSection
		var err error
		if sec.value, err = parseAmount(vals[0]); err != nil {
			return portfolioSection{}, fmt.Errorf("%s value: %w", key, err)
		}
		if sec.pnl, err = parseAmount(vals[1]); err != nil {
			return portfolioSection{}, fmt.Errorf("%s pnl: %w", key, err)
		}
		if sec.pnlPct, err = parseAmount(vals[2]); err != nil {
			return portfolioSection{}, fmt.Errorf("%s pnl%%: %w", key, err)
		}
		if sec.prevDay, err = parseAmount(vals[3]); err != nil {
			return portfolioSection{}, fmt.Errorf("%s prev day: %w", key, err)
		}
		if sec.prevDayPct, err = parseAmount(vals[4]); err != nil {
			return portfolioSection{}, fmt.Errorf("%s prev day%%: %w", key, err)
		}
		return sec, nil
	}
	return portfolioSection{}, fmt.Errorf("section %q not found", key)
}

// parseForeignCash extracts the USD 預り金 from the foreign 口座サマリー
// body text (米ドル row of the 保有資産評価 table).
func parseForeignCash(text string) (Foreign, error) {
	ls := lines(text)
	for i, l := range ls {
		if l == "米ドル" && i+2 < len(ls) {
			usd, err := parseAmount(ls[i+1])
			if err != nil {
				return Foreign{}, err
			}
			jpy, err := parseAmount(ls[i+2])
			if err != nil {
				return Foreign{}, err
			}
			return Foreign{USD: usd, JPY: jpy}, nil
		}
	}
	return Foreign{}, fmt.Errorf("米ドル deposit row not found")
}

// parseNISA extracts the summary block of the NISA portfolio page:
// total balance, prev-day/prev-month/P&L for the whole account, and
// the per-asset-class breakdown (国内株式 / 米国株式 / 投資信託).
func parseNISA(text string) (NISA, error) {
	ls := lines(text)
	var nisa NISA
	var err error
	if nisa.TotalJPY, err = valueAfter(ls, "NISA資産残高"); err != nil {
		return NISA{}, err
	}
	if nisa.PrevDayJPY, nisa.PrevDayPct, err = pairAfter(ls, "前日比/率"); err != nil {
		return NISA{}, err
	}
	if nisa.PrevMonthJPY, nisa.PrevMonthPct, err = pairAfter(ls, "前月比/率"); err != nil {
		return NISA{}, err
	}
	if nisa.PnLJPY, nisa.PnLPct, err = pairAfter(ls, "評価損益/率"); err != nil {
		return NISA{}, err
	}
	for _, key := range []string{"国内株式", "米国株式", "投資信託"} {
		item, err := parseNISAItem(ls, key)
		if err != nil {
			return NISA{}, err
		}
		switch key {
		case "国内株式":
			nisa.Domestic = item
		case "米国株式":
			nisa.USStocks = item
		case "投資信託":
			nisa.Funds = item
		}
	}
	return nisa, nil
}

// parseNISAItem parses one asset-class breakdown section. The section
// header line (e.g. 国内株式) is followed by 評価額 / 評価損益 / 前日比 /
// 前月比 pairs. The header is only matched when 評価額 follows it, which
// distinguishes it from identical labels in the nav bar.
func parseNISAItem(ls []string, key string) (NISAItem, error) {
	start := -1
	for i := 0; i+1 < len(ls); i++ {
		if ls[i] == key && ls[i+1] == "評価額" {
			start = i + 1
			break
		}
	}
	if start < 0 {
		return NISAItem{}, fmt.Errorf("%s section not found", key)
	}
	seg := ls[start:]
	var item NISAItem
	var err error
	if item.ValueJPY, err = valueAfter(seg, "評価額"); err != nil {
		return NISAItem{}, err
	}
	if item.PnLJPY, item.PnLPct, err = pairAfter(seg, "評価損益"); err != nil {
		return NISAItem{}, err
	}
	if item.PrevDayJPY, item.PrevDayPct, err = pairAfter(seg, "前日比"); err != nil {
		return NISAItem{}, err
	}
	if item.PrevMonthJPY, item.PrevMonthPct, err = pairAfter(seg, "前月比"); err != nil {
		return NISAItem{}, err
	}
	return item, nil
}

// pairAfter returns the amount on the line after the key line and the
// percentage on the line after that (e.g. "3,771,642円" then
// "(+34.49%)").
func pairAfter(ls []string, key string) (amount, pct float64, err error) {
	for i, l := range ls {
		if l == key && i+2 < len(ls) {
			if amount, err = parseAmount(ls[i+1]); err != nil {
				return 0, 0, err
			}
			if pct, err = parseAmount(ls[i+2]); err != nil {
				return 0, 0, err
			}
			return amount, pct, nil
		}
	}
	return 0, 0, fmt.Errorf("key %q not found", key)
}
