package nrkn

import (
	"context"
	"fmt"
	"strings"
	"unicode"

	"github.com/azuki774/myscrapers/myscraper/internal/figi"
)

// HoldingFIGIResolver maps NRKN product codes to composite FIGIs. Product codes
// are local keys only; they must never be sent to OpenFIGI as market tickers.
type HoldingFIGIResolver interface {
	Resolve(context.Context, []Holding) (map[string]string, error)
}

type holdingFIGIResolver struct{ client figi.Resolver }

func NewHoldingFIGIResolver(client figi.Resolver) HoldingFIGIResolver {
	if client == nil {
		client = figi.NewOpenFIGIResolver(nil)
	}
	return &holdingFIGIResolver{client: client}
}

// Verified public fund identifiers, not a list of account holdings. Add funds
// only after checking the manager's source and the exact NRKN display name.
// FIGIs themselves are resolved on each run, using the same client as SBI.
var fundIdentifiers = []struct {
	ticker string
	names  []string
	source string
}{
	{
		ticker: "29316149",
		names:  []string{"DCニッセイ国内株式インデックス"},
		source: "https://www.nam.co.jp/fundinfo/dcnkki/main.html",
	},
	{
		ticker: "01312022",
		names: []string{
			"野村外株インデックスファンド・MSCI-KOKUSAI・DC",
			"野村外国株式インデックスファンド・MSCI-KOKUSAI（確定拠出年金向け）",
		},
		source: "https://www.nomura-am.co.jp/news/20160928_1D8BFFA4.pdf",
	},
	{
		ticker: "01314027",
		names: []string{
			"マイバランス70（確定拠出年金向け）・野村",
			"マイバランス70（確定拠出年金向け）",
		},
		source: "https://www.nomura-am.co.jp/news/20160928_1D8BFFA4.pdf",
	},
}

func normalizeFundName(name string) string {
	return strings.Map(func(r rune) rune {
		if unicode.IsSpace(r) {
			return -1
		}
		if r >= 0xff01 && r <= 0xff5e {
			r -= 0xfee0
		}
		return unicode.ToUpper(r)
	}, name)
}

func fundLookup(h Holding) (figi.FIGILookup, error) {
	if h.ProductCode == "" || normalizeFundName(h.Category) != "国内投信" {
		return figi.FIGILookup{}, fmt.Errorf("nrkn: unsupported FIGI product %q", h.ProductCode)
	}
	name := normalizeFundName(h.Name)
	var ticker string
	for _, fund := range fundIdentifiers {
		for _, alias := range fund.names {
			if normalizeFundName(alias) == name {
				if ticker != "" && ticker != fund.ticker {
					return figi.FIGILookup{}, fmt.Errorf("nrkn: ambiguous fund identifier for product %q", h.ProductCode)
				}
				ticker = fund.ticker
			}
		}
	}
	if ticker == "" {
		return figi.FIGILookup{}, fmt.Errorf("nrkn: no verified fund identifier for product %q", h.ProductCode)
	}
	return figi.FIGILookup{Ticker: ticker, ExchCode: "JP"}, nil
}

func (r *holdingFIGIResolver) Resolve(ctx context.Context, holdings []Holding) (map[string]string, error) {
	lookups := make([]figi.FIGILookup, len(holdings))
	seen := make(map[string]bool, len(holdings))
	for i, h := range holdings {
		if seen[h.ProductCode] {
			return nil, fmt.Errorf("nrkn: duplicate FIGI product %q", h.ProductCode)
		}
		seen[h.ProductCode] = true
		lookup, err := fundLookup(h)
		if err != nil {
			return nil, err
		}
		lookups[i] = lookup
	}
	resolved, err := r.client.Resolve(ctx, lookups)
	if err != nil {
		return nil, fmt.Errorf("nrkn: resolve FIGIs: %w", err)
	}
	result := make(map[string]string, len(holdings))
	for i, h := range holdings {
		value := resolved[lookups[i]]
		if strings.TrimSpace(value) == "" {
			return nil, fmt.Errorf("nrkn: missing composite FIGI for product %q", h.ProductCode)
		}
		result[h.ProductCode] = value
	}
	return result, nil
}
