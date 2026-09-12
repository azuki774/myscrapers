package sbi

import (
	"fmt"
	"net/url"
	"regexp"
	"strings"
	"unicode"

	"golang.org/x/net/html"
)

var domesticSecurityCodeRe = regexp.MustCompile(`^[A-Za-z0-9]{4}$`)
var fundSecurityCodeRe = regexp.MustCompile(`^[A-Za-z0-9]{8}$`)
var usTickerRe = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9.\-]*$`)

// parseStockRowsStrict validates the source security code retained by the
// ordinary row parser. The four-character SBI code is the JP lookup ticker.
func parseStockRowsStrict(tokens []string) ([]Holding, error) {
	stocks := parseStockRows(tokens)
	markers := 0
	for i := 0; i+2 < len(tokens); i++ {
		if tokens[i] == "現買" && tokens[i+1] == "現売" && tokens[i+2] == "積立" {
			markers++
		}
	}
	if len(stocks) != markers {
		return nil, fmt.Errorf("parsed %d of %d domestic holding rows", len(stocks), markers)
	}
	for _, stock := range stocks {
		if stock.source == nil || !domesticSecurityCodeRe.MatchString(stock.source.Ticker) {
			return nil, fmt.Errorf("holding %q has invalid or missing domestic security code", stock.Name)
		}
	}
	return stocks, nil
}

func parseUSHoldingsStrict(text string) ([]Holding, error) {
	holdings := parseUSHoldings(text)
	markers := 0
	for _, token := range strings.Fields(text) {
		if token == "(0)" {
			markers++
		}
	}
	if len(holdings) != markers {
		return nil, fmt.Errorf("parsed %d of %d US holding rows", len(holdings), markers)
	}
	for _, holding := range holdings {
		if holding.source == nil || holding.source.Ticker == "" {
			return nil, fmt.Errorf("holding %q has missing US ticker", holding.Name)
		}
	}
	return holdings, nil
}

func tickerFromUSToken(token string) string {
	markets := []string{"NYSEAMERICAN", "NYSEARCA", "NASDAQ", "NYSE", "CBOE", "OTC"}
	for _, market := range markets {
		if ticker, ok := removeMarketSuffix(token, market); ok {
			if usTickerRe.MatchString(ticker) {
				return ticker
			}
			return ""
		}
	}
	return ""
}

// removeMarketSuffix normalizes only the market suffix. In particular, it
// preserves ticker punctuation such as the hyphen in BRK-B.
func removeMarketSuffix(token, market string) (string, bool) {
	tokenRunes := []rune(strings.ToUpper(strings.TrimSpace(token)))
	marketRunes := []rune(market)
	i, j := len(tokenRunes)-1, len(marketRunes)-1
	for j >= 0 {
		for i >= 0 && unicode.IsSpace(tokenRunes[i]) {
			i--
		}
		if i < 0 || tokenRunes[i] != marketRunes[j] {
			return "", false
		}
		i--
		j--
	}
	return strings.TrimSpace(string(tokenRunes[:i+1])), true
}

type fundHTMLCandidate struct {
	name string
	code string
}

// parseFundRowsStrict attaches fund_sec_code values from rendered portfolio
// HTML. Matching is by normalized holding name and must yield one code.
func parseFundRowsStrict(tokens []string, pageHTML string) ([]Holding, error) {
	funds := parseFundRows(tokens)
	markers := 0
	for i := 0; i+1 < len(tokens); i++ {
		if tokens[i] == "積立" && tokens[i+1] == "売却" {
			markers++
		}
	}
	if len(funds) != markers {
		return nil, fmt.Errorf("parsed %d of %d fund holding rows", len(funds), markers)
	}
	if len(funds) == 0 {
		return funds, nil
	}
	candidates, err := fundHTMLCandidates(pageHTML)
	if err != nil {
		return nil, err
	}
	for i := range funds {
		name := normalizeHoldingName(funds[i].Name)
		codes := make(map[string]struct{})
		for _, candidate := range candidates {
			if candidate.name == name || strings.Contains(candidate.name, name) {
				codes[candidate.code] = struct{}{}
			}
		}
		if len(codes) != 1 {
			return nil, fmt.Errorf("fund %q matched %d fund_sec_code values", funds[i].Name, len(codes))
		}
		for code := range codes {
			funds[i].source = &FIGILookup{Ticker: strings.ToUpper(code), ExchCode: "JP"}
		}
	}
	return funds, nil
}

func normalizeHoldingName(name string) string {
	var b strings.Builder
	space := false
	for _, r := range name {
		switch {
		case r == '　' || unicode.IsSpace(r):
			space = true
		case r >= 0xff01 && r <= 0xff5e:
			if space && b.Len() > 0 {
				b.WriteByte(' ')
			}
			space = false
			b.WriteRune(unicode.ToLower(r - 0xfee0))
		default:
			if space && b.Len() > 0 {
				b.WriteByte(' ')
			}
			space = false
			b.WriteRune(unicode.ToLower(r))
		}
	}
	return strings.TrimSpace(b.String())
}

func fundHTMLCandidates(pageHTML string) ([]fundHTMLCandidate, error) {
	if strings.TrimSpace(pageHTML) == "" {
		return nil, fmt.Errorf("portfolio HTML is empty; cannot resolve fund identifiers")
	}
	doc, err := html.Parse(strings.NewReader(pageHTML))
	if err != nil {
		return nil, fmt.Errorf("parse portfolio HTML: %w", err)
	}
	var candidates []fundHTMLCandidate
	var walk func(*html.Node)
	walk = func(node *html.Node) {
		if node.Type == html.ElementNode && node.Data == "tr" {
			text := normalizeHoldingName(nodeText(node))
			for _, code := range linksWithFundCode(node) {
				candidates = append(candidates, fundHTMLCandidate{name: text, code: code})
			}
			return
		}
		for child := node.FirstChild; child != nil; child = child.NextSibling {
			walk(child)
		}
	}
	walk(doc)
	return candidates, nil
}

func nodeText(node *html.Node) string {
	if node.Type == html.TextNode {
		return node.Data
	}
	var b strings.Builder
	for child := node.FirstChild; child != nil; child = child.NextSibling {
		b.WriteString(nodeText(child))
		b.WriteByte(' ')
	}
	return b.String()
}

func linksWithFundCode(node *html.Node) []string {
	seen := make(map[string]struct{})
	var codes []string
	var walk func(*html.Node)
	walk = func(n *html.Node) {
		if n.Type == html.ElementNode && n.Data == "a" {
			for _, attr := range n.Attr {
				if attr.Key != "href" {
					continue
				}
				u, err := url.Parse(attr.Val)
				if err != nil {
					continue
				}
				code := u.Query().Get("fund_sec_code")
				if fundSecurityCodeRe.MatchString(code) {
					if _, ok := seen[code]; !ok {
						seen[code] = struct{}{}
						codes = append(codes, code)
					}
				}
			}
		}
		for child := n.FirstChild; child != nil; child = child.NextSibling {
			walk(child)
		}
	}
	walk(node)
	return codes
}
