package nrkn

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"math"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/azuki774/myscrapers/myscraper/internal/logging"
	"golang.org/x/net/html"
)

const CurrentSchemaVersion = "2026-09-14"
const StatusOK = "ok"

type FetchOptions struct {
	Credentials        Credentials
	OutputPath         string
	Now                time.Time
	Logger             *slog.Logger
	Headless, S3Upload bool
	S3Client           S3Client
	FIGIResolver       HoldingFIGIResolver
}
type S3Client interface {
	PutJSON(context.Context, string, io.Reader) error
	KeyForTime(time.Time) string
}

type Assets struct {
	SchemaVersion string    `json:"schema_version"`
	FetchedAt     time.Time `json:"fetched_at"`
	Status        string    `json:"status"`
	GrandTotalJPY int64     `json:"grand_total_jpy"`
	TotalCostJPY  int64     `json:"total_cost_jpy"`
	PnLJPY        int64     `json:"pnl_jpy"`
	Holdings      []Holding `json:"holdings"`
}
type Holding struct {
	CompositeFIGI          string  `json:"composite_figi"`
	ProductCode            string  `json:"product_code"`
	Name                   string  `json:"name"`
	Category               string  `json:"category"`
	Quantity               float64 `json:"quantity"`
	UnitPrice              float64 `json:"unit_price"`
	ValueJPY               int64   `json:"value_jpy"`
	CostJPY                int64   `json:"cost_jpy"`
	RedemptionUnitPrice    float64 `json:"redemption_unit_price"`
	RedemptionValueJPY     int64   `json:"redemption_value_jpy"`
	PnLJPY                 int64   `json:"pnl_jpy"`
	ReferenceDate          string  `json:"reference_date"`
	AllocationPct          float64 `json:"allocation_pct"`
	UnitPriceRaw           string  `json:"unit_price_raw"`
	RedemptionUnitPriceRaw string  `json:"redemption_unit_price_raw"`
	parsedMask             uint16
}

func FetchAssets(ctx context.Context, sess Session, opts FetchOptions) (assets *Assets, retErr error) {
	if sess == nil {
		return nil, fmt.Errorf("nrkn: session is required")
	}
	if opts.FIGIResolver == nil {
		return nil, fmt.Errorf("nrkn: FIGI resolver is required")
	}
	log := logging.New(opts.Logger, "nrkn")
	defer func() {
		cleanupStarted := log.Started("cleanup")
		cleanupCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := sess.Logout(cleanupCtx); err != nil {
			log.Warning("cleanup", "logout failed", "operation", "logout", "error", err)
			if retErr == nil {
				retErr = logging.WithContext(fmt.Errorf("logout: %w", err), "cleanup", "")
			} else {
				retErr = fmt.Errorf("%w; logout: %v", retErr, err)
			}
		} else {
			log.Completed("cleanup", cleanupStarted, "operation", "logout")
		}
	}()

	loginStarted := log.Started("login")
	if err := sess.Login(ctx, opts.Credentials.ID, opts.Credentials.Password, opts.Credentials.Birthday); errors.Is(err, ErrConcurrentLogin) {
		log.Warning("login", "retrying", "reason", "concurrent_login", "attempt", 2)
		if err = sess.Login(ctx, opts.Credentials.ID, opts.Credentials.Password, opts.Credentials.Birthday); err != nil {
			return nil, logging.WithContext(fmt.Errorf("login retry: %w", err), "login", "")
		}
	} else if err != nil {
		return nil, logging.WithContext(fmt.Errorf("login: %w", err), "login", "")
	}
	log.Completed("login", loginStarted)
	// Logout is attempted exactly once by the orchestrator. A cleanup failure
	// is returned after the JSON has been assembled so callers can preserve it
	// while still reporting a failed run.
	pageStarted := log.Started("page", "page", "asset_valuation")
	pageFail := func(err error) error {
		return logging.WithContext(err, "page", "asset_valuation")
	}
	if err := sess.NavigateToAssets(ctx); err != nil {
		return nil, pageFail(err)
	}
	htmlText, err := sess.BodyHTML(ctx)
	if err != nil {
		return nil, pageFail(err)
	}
	holdings, err := ParseHoldingsHTML(htmlText)
	if err != nil {
		return nil, pageFail(err)
	}
	if len(holdings) == 0 {
		return nil, pageFail(fmt.Errorf("nrkn: no holdings found"))
	}
	a := &Assets{SchemaVersion: CurrentSchemaVersion, FetchedAt: opts.Now, Status: StatusOK, Holdings: holdings}
	totals, err := parseTotalsHTML(htmlText)
	if err != nil {
		return nil, pageFail(err)
	}
	a.GrandTotalJPY, a.TotalCostJPY, a.PnLJPY = totals[0], totals[1], totals[2]
	var sums [3]int64
	for _, h := range holdings {
		sums[0] += h.ValueJPY
		sums[1] += h.CostJPY
		sums[2] += h.PnLJPY
	}
	if sums != totals {
		log.Warning("page", "displayed totals differ from holding sums; preserving displayed values", "page", "asset_valuation")
	}
	log.Completed("page", pageStarted, "page", "asset_valuation", "holdings", len(holdings))

	resolveStarted := log.Started("resolve_figi", "count", len(holdings))
	resolved, err := opts.FIGIResolver.Resolve(ctx, holdings)
	if err != nil {
		return nil, logging.WithContext(fmt.Errorf("nrkn: resolve FIGIs: %w", err), "resolve_figi", "")
	}
	for i := range a.Holdings {
		figi := strings.TrimSpace(resolved[a.Holdings[i].ProductCode])
		if figi == "" {
			return nil, logging.WithContext(fmt.Errorf("nrkn: missing composite FIGI for product %q", a.Holdings[i].ProductCode), "resolve_figi", "")
		}
		a.Holdings[i].CompositeFIGI = figi
	}
	log.Completed("resolve_figi", resolveStarted, "count", len(holdings))
	return a, nil
}

func ParseHoldingsHTML(source string) ([]Holding, error) {
	doc, err := html.Parse(strings.NewReader(source))
	if err != nil {
		return nil, errors.New("nrkn: invalid HTML")
	}
	var result []Holding
	seen := map[string]bool{}
	var current *Holding
	var parseErr error
	finish := func() {
		if current == nil || parseErr != nil {
			return
		}
		if current.ProductCode == "" || current.Name == "" || current.Category == "" || current.parsedMask != 511 {
			parseErr = errors.New("nrkn: incomplete product block")
			return
		}
		if seen[current.ProductCode] {
			parseErr = errors.New("nrkn: duplicate product code")
			return
		}
		seen[current.ProductCode] = true
		result = append(result, *current)
		current = nil
	}
	var walk func(*html.Node)
	walk = func(n *html.Node) {
		if parseErr != nil || hidden(n) {
			return
		}
		if n.Type == html.ElementNode && hasClass(n, "infoHdWrap") && strings.Contains(visibleText(n), "商品名") {
			finish()
			h := heading(n)
			current = &h
		}
		if n.Type == html.ElementNode && n.Data == "table" {
			if current != nil {
				if err := applyTable(current, n); err != nil {
					parseErr = err
				}
			}
			return
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
	}
	walk(doc)
	finish()
	if parseErr != nil {
		return nil, parseErr
	}
	if len(result) == 0 {
		return nil, errors.New("nrkn: no product blocks found")
	}
	return result, nil
}
func attr(n *html.Node, key string) string {
	for _, a := range n.Attr {
		if a.Key == key {
			return a.Val
		}
	}
	return ""
}
func hasClass(n *html.Node, name string) bool {
	for _, c := range strings.Fields(attr(n, "class")) {
		if c == name {
			return true
		}
	}
	return false
}
func hidden(n *html.Node) bool {
	if n.Type != html.ElementNode {
		return false
	}
	if n.Data == "script" || n.Data == "style" || n.Data == "noscript" {
		return true
	}
	style := strings.ReplaceAll(strings.ToLower(attr(n, "style")), " ", "")
	for _, a := range n.Attr {
		if a.Key == "hidden" {
			return true
		}
	}
	return attr(n, "lang") == "en" || attr(n, "data-lang") == "en" || attr(n, "data-lang") == "eng" || strings.Contains(style, "display:none") || hasClass(n, "en") || hasClass(n, "english")
}
func visibleText(n *html.Node) string {
	if hidden(n) {
		return ""
	}
	if n.Type == html.TextNode {
		return n.Data
	}
	var b strings.Builder
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		b.WriteString(" ")
		b.WriteString(visibleText(c))
	}
	return strings.Join(strings.Fields(b.String()), " ")
}
func heading(n *html.Node) Holding {
	var h Holding
	var walk func(*html.Node)
	walk = func(x *html.Node) {
		if hidden(x) {
			return
		}
		if x.Type == html.ElementNode && x.Data == "dt" {
			label := visibleText(x)
			for dd := x.NextSibling; dd != nil; dd = dd.NextSibling {
				if dd.Type != html.ElementNode {
					continue
				}
				if dd.Data != "dd" {
					break
				}
				v := visibleText(dd)
				switch label {
				case "商品名":
					h.Name = v
				case "商品コード":
					h.ProductCode = v
				case "商品分類":
					h.Category = v
				}
				break
			}
		}
		for c := x.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
	}
	walk(n)
	return h
}
func tableRows(n *html.Node) [][]string {
	var rows [][]string
	var walk func(*html.Node)
	walk = func(x *html.Node) {
		if hidden(x) {
			return
		}
		if x.Type == html.ElementNode && x.Data == "tr" {
			var cells []string
			nonempty := false
			for c := x.FirstChild; c != nil; c = c.NextSibling {
				if c.Type == html.ElementNode && (c.Data == "th" || c.Data == "td") {
					v := visibleText(c)
					cells = append(cells, v)
					nonempty = nonempty || v != ""
				}
			}
			if nonempty {
				rows = append(rows, cells)
			}
			return
		}
		for c := x.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
	}
	walk(n)
	return rows
}

var numberPattern = regexp.MustCompile(`^[+-]?(?:[0-9]+|[0-9]{1,3}(?:,[0-9]{3})+)(?:\.[0-9]+)?$`)

func parseNumber(v string) (float64, error) {
	v = strings.TrimSpace(v)
	v = strings.NewReplacer("，", ",", "％", "%", "−", "-", "－", "-", "▲", "-", "△", "-").Replace(v)
	for _, suffix := range []string{"円", "%"} {
		v = strings.TrimSpace(strings.TrimSuffix(v, suffix))
	}
	if !numberPattern.MatchString(v) {
		return 0, errors.New("nrkn: invalid numeric value")
	}
	n, err := strconv.ParseFloat(strings.ReplaceAll(v, ",", ""), 64)
	if err != nil || math.IsNaN(n) || math.IsInf(n, 0) {
		return 0, errors.New("nrkn: invalid numeric value")
	}
	return n, nil
}
func parseMoney(v string) (int64, error) {
	n, err := parseNumber(v)
	if err != nil || n != math.Trunc(n) || n >= math.MaxInt64 || n <= math.MinInt64 {
		return 0, errors.New("nrkn: invalid integer amount")
	}
	return int64(n), nil
}
func applyTable(h *Holding, n *html.Node) error {
	rows := tableRows(n)
	for row, hdr := range rows {
		for col, k := range hdr {
			bit := uint16(0)
			switch k {
			case "数量（残高）", "数量(残高)":
				bit = 1
			case "基準価額":
				bit = 2
			case "資産評価額":
				bit = 4
			case "取得価額累計":
				bit = 8
			case "解約価額":
				bit = 16
			case "解約時評価額":
				bit = 32
			case "損益":
				bit = 64
			case "基準日":
				bit = 128
			case "資産比率":
				bit = 256
			}
			if bit == 0 {
				continue
			}
			if row+1 >= len(rows) || col >= len(rows[row+1]) {
				return fmt.Errorf("nrkn: missing %s", k)
			}
			if h.parsedMask&bit != 0 {
				return fmt.Errorf("nrkn: duplicate %s", k)
			}
			v := rows[row+1][col]
			var err error
			switch bit {
			case 1:
				h.Quantity, err = parseNumber(v)
			case 2:
				h.UnitPriceRaw = v
				h.UnitPrice, err = parseNumber(strings.TrimSpace(strings.TrimPrefix(v, "*")))
			case 4:
				h.ValueJPY, err = parseMoney(v)
			case 8:
				h.CostJPY, err = parseMoney(v)
			case 16:
				h.RedemptionUnitPriceRaw = v
				h.RedemptionUnitPrice, err = parseNumber(strings.TrimSpace(strings.TrimPrefix(v, "*")))
			case 32:
				h.RedemptionValueJPY, err = parseMoney(v)
			case 64:
				h.PnLJPY, err = parseMoney(v)
			case 128:
				var date time.Time
				date, err = time.Parse("2006/01/02", v)
				if err != nil {
					date, err = time.Parse("2006-01-02", v)
				}
				if err == nil {
					h.ReferenceDate = date.Format("2006-01-02")
				}
			case 256:
				h.AllocationPct, err = parseNumber(v)
				if h.AllocationPct < 0 || h.AllocationPct > 100 {
					err = errors.New("invalid ratio")
				}
			}
			if err != nil {
				return fmt.Errorf("nrkn: invalid %s", k)
			}
			h.parsedMask |= bit
		}
	}
	return nil
}
func parseTotalsHTML(source string) ([3]int64, error) {
	var totals [3]int64
	var found [3]bool
	doc, err := html.Parse(strings.NewReader(source))
	if err != nil {
		return totals, errors.New("nrkn: invalid HTML")
	}
	var parseErr error
	var walk func(*html.Node)
	walk = func(n *html.Node) {
		if hidden(n) || parseErr != nil {
			return
		}
		if n.Type == html.ElementNode && n.Data == "table" {
			rows := tableRows(n)
			for ri, r := range rows {
				for ci, label := range r {
					index := -1
					switch strings.ReplaceAll(label, " ", "") {
					case "資産評価額合計":
						index = 0
					case "取得価額累計合計":
						index = 1
					case "損益合計":
						index = 2
					}
					if index < 0 {
						continue
					}
					if found[index] || ri+1 >= len(rows) || ci >= len(rows[ri+1]) {
						parseErr = errors.New("nrkn: invalid totals table")
						return
					}
					totals[index], err = parseMoney(rows[ri+1][ci])
					if err != nil {
						parseErr = errors.New("nrkn: invalid total amount")
						return
					}
					found[index] = true
				}
			}
			return
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
	}
	walk(doc)
	if parseErr != nil {
		return totals, parseErr
	}
	if found != [3]bool{true, true, true} {
		return totals, errors.New("nrkn: missing displayed totals")
	}
	return totals, nil
}
