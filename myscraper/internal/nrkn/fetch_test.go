package nrkn

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"strings"
	"testing"
	"time"
)

type fakeSession struct {
	loginCalls, navigateCalls, bodyCalls, logoutCalls int
	loginErrs                                         []error
	body                                              string
	logoutErr                                         error
}

type fakeHoldingResolver struct {
	err    error
	values map[string]string
}

func (f fakeHoldingResolver) Resolve(_ context.Context, holdings []Holding) (map[string]string, error) {
	if f.err != nil {
		return nil, f.err
	}
	if f.values != nil {
		return f.values, nil
	}
	r := make(map[string]string, len(holdings))
	for _, h := range holdings {
		r[h.ProductCode] = "FIGI-" + h.ProductCode
	}
	return r, nil
}

func (f *fakeSession) Login(context.Context, string, string, string) error {
	i := f.loginCalls
	f.loginCalls++
	if i < len(f.loginErrs) {
		return f.loginErrs[i]
	}
	return nil
}
func (f *fakeSession) NavigateToAssets(context.Context) error   { f.navigateCalls++; return nil }
func (f *fakeSession) BodyHTML(context.Context) (string, error) { f.bodyCalls++; return f.body, nil }
func (f *fakeSession) Logout(context.Context) error             { f.logoutCalls++; return f.logoutErr }
func (f *fakeSession) Close() error                             { return nil }

func testOptions() FetchOptions {
	return FetchOptions{Credentials: Credentials{ID: "id", Password: "pass", Birthday: "20000101"}, Now: time.Unix(0, 0), FIGIResolver: fakeHoldingResolver{}}
}

func TestParseHoldingsHTMLMapsJapaneseTableRows(t *testing.T) {
	const fixture = `<div class="infoHdWrap"><dl><dt>商品名</dt><dd>架空国内株式インデックス</dd><dt>商品コード</dt><dd>09999</dd><dt>商品分類</dt><dd>国内投信</dd></dl></div><table><tr><th>数量（残高）</th><th>基準価額</th><th>資産評価額</th><th>取得価額累計</th></tr><tr lang="en"><th>Quantity</th><th>Price</th><th>Value</th><th>Cost</th></tr><tr><td>12,345</td><td>12,345円</td><td>234,567円</td><td>200,000円</td></tr><tr><th>解約価額</th><th>解約時評価額</th><th>損益</th></tr><tr><td>12,300円</td><td>230,000円</td><td>-1,234円</td></tr><tr><th>基準日</th><th>資産比率</th></tr><tr><td>2026/09/11</td><td>34％</td></tr></table>`
	h, err := ParseHoldingsHTML(fixture)
	if err != nil {
		t.Fatal(err)
	}
	if len(h) != 1 {
		t.Fatalf("holdings=%d", len(h))
	}
	g := h[0]
	if g.ProductCode != "09999" || g.Name != "架空国内株式インデックス" || g.Category != "国内投信" {
		t.Errorf("heading=%+v", g)
	}
	if g.Quantity != 12345 || g.UnitPrice != 12345 || g.ValueJPY != 234567 || g.CostJPY != 200000 || g.RedemptionValueJPY != 230000 || g.PnLJPY != -1234 {
		t.Errorf("values=%+v", g)
	}
	if g.ReferenceDate != "2026-09-11" || g.AllocationPct != 34 {
		t.Errorf("date/pct=%+v", g)
	}
}

func TestFetchAssetsLoginFailureDoesNotRetryAndLogsOut(t *testing.T) {
	f := &fakeSession{loginErrs: []error{errors.New("authentication failed")}}
	_, err := FetchAssets(context.Background(), f, testOptions())
	if err == nil || f.loginCalls != 1 || f.logoutCalls != 1 {
		t.Fatalf("err=%v login=%d logout=%d", err, f.loginCalls, f.logoutCalls)
	}
}

func TestFetchAssetsConcurrentLoginRetriesOnceOnly(t *testing.T) {
	f := &fakeSession{loginErrs: []error{ErrConcurrentLogin, ErrConcurrentLogin}}
	_, err := FetchAssets(context.Background(), f, testOptions())
	if err == nil || f.loginCalls != 2 || f.logoutCalls != 1 {
		t.Fatalf("err=%v login=%d logout=%d", err, f.loginCalls, f.logoutCalls)
	}
}

func TestFetchAssetsParseErrorStillLogsOut(t *testing.T) {
	f := &fakeSession{body: "<html><body>unexpected</body></html>"}
	_, err := FetchAssets(context.Background(), f, testOptions())
	if err == nil || f.loginCalls != 1 || f.navigateCalls != 1 || f.bodyCalls != 1 || f.logoutCalls != 1 {
		t.Fatalf("err=%v calls login=%d navigate=%d body=%d logout=%d", err, f.loginCalls, f.navigateCalls, f.bodyCalls, f.logoutCalls)
	}
}

func TestFetchAssetsCleanupFailureReturnsAssetsAndError(t *testing.T) {
	const fixture = `<div class="infoHdWrap"><dl><dt>商品名</dt><dd>架空ファンド</dd><dt>商品コード</dt><dd>08888</dd><dt>商品分類</dt><dd>国内投信</dd></dl></div><table><tr><th>数量（残高）</th><th>基準価額</th><th>資産評価額</th><th>取得価額累計</th></tr><tr><td>1</td><td>100円</td><td>100円</td><td>90円</td></tr><tr><th>解約価額</th><th>解約時評価額</th><th>損益</th></tr><tr><td>100円</td><td>100円</td><td>10円</td></tr><tr><th>基準日</th><th>資産比率</th></tr><tr><td>2026/09/11</td><td>100％</td></tr></table><table><tr><th>資産評価額合計</th><th>取得価額累計合計</th><th>損益合計</th></tr><tr><td>100円</td><td>90円</td><td>10円</td></tr></table>`
	f := &fakeSession{body: fixture, logoutErr: errors.New("logout failed")}
	a, err := FetchAssets(context.Background(), f, testOptions())
	if err == nil || a == nil || f.logoutCalls != 1 {
		t.Fatalf("assets=%v err=%v logout=%d", a, err, f.logoutCalls)
	}
}

func resolverFixture() string {
	return `<div class="infoHdWrap"><dl><dt>商品名</dt><dd>架空</dd><dt>商品コード</dt><dd>06666</dd><dt>商品分類</dt><dd>国内投信</dd></dl></div><table><tr><th>数量（残高）</th><th>基準価額</th><th>資産評価額</th><th>取得価額累計</th></tr><tr><td>1</td><td>100円</td><td>100円</td><td>90円</td></tr><tr><th>解約価額</th><th>解約時評価額</th><th>損益</th></tr><tr><td>100円</td><td>100円</td><td>10円</td></tr><tr><th>基準日</th><th>資産比率</th></tr><tr><td>2026/09/11</td><td>100％</td></tr></table><table><tr><th>資産評価額合計</th><th>取得価額累計合計</th><th>損益合計</th></tr><tr><td>100円</td><td>90円</td><td>10円</td></tr></table>`
}

func TestFetchAssetsFIGIResolverFailureReturnsNoAssetsAndLogsOut(t *testing.T) {
	f := &fakeSession{body: resolverFixture()}
	o := testOptions()
	o.FIGIResolver = fakeHoldingResolver{err: errors.New("resolver unavailable")}
	a, err := FetchAssets(context.Background(), f, o)
	if err == nil || a != nil || f.logoutCalls != 1 {
		t.Fatalf("assets=%v err=%v logout=%d", a, err, f.logoutCalls)
	}
}

func TestFetchAssetsFIGIResolverPartialMapReturnsNoAssets(t *testing.T) {
	second := strings.ReplaceAll(resolverFixture(), "06666", "05555")
	second = strings.SplitN(second, "<table><tr><th>資産評価額合計", 2)[0]
	f := &fakeSession{body: resolverFixture() + second}
	o := testOptions()
	o.FIGIResolver = fakeHoldingResolver{values: map[string]string{"06666": "BBGTESTFUND"}}
	a, err := FetchAssets(context.Background(), f, o)
	if err == nil || a != nil || f.logoutCalls != 1 {
		t.Fatalf("assets=%v err=%v logout=%d", a, err, f.logoutCalls)
	}
}

func TestFetchAssetsAttachesFIGIToEveryHolding(t *testing.T) {
	second := strings.ReplaceAll(resolverFixture(), "06666", "05555")
	second = strings.SplitN(second, "<table><tr><th>資産評価額合計", 2)[0]
	f := &fakeSession{body: resolverFixture() + second}
	a, err := FetchAssets(context.Background(), f, testOptions())
	if err != nil || a == nil {
		t.Fatalf("fetch failed: %v", err)
	}
	if len(a.Holdings) != 2 || f.logoutCalls != 1 {
		t.Fatalf("holdings=%d logout=%d", len(a.Holdings), f.logoutCalls)
	}
	for _, h := range a.Holdings {
		if h.CompositeFIGI != "FIGI-"+h.ProductCode {
			t.Fatalf("missing or misplaced FIGI for %s", h.ProductCode)
		}
	}
}

func TestFetchAssetsRequiresFIGIResolverBeforeLogin(t *testing.T) {
	f := &fakeSession{body: resolverFixture()}
	o := testOptions()
	o.FIGIResolver = nil
	a, err := FetchAssets(context.Background(), f, o)
	if err == nil || a != nil || f.loginCalls != 0 || f.logoutCalls != 0 {
		t.Fatalf("assets=%v err=%v login=%d logout=%d", a, err, f.loginCalls, f.logoutCalls)
	}
}

func TestParseHoldingsHTMLRejectsMalformedNumeric(t *testing.T) {
	const fixture = `<div class="infoHdWrap"><dl><dt>商品名</dt><dd>架空</dd><dt>商品コード</dt><dd>07777</dd><dt>商品分類</dt><dd>国内投信</dd></dl></div><table><tr><th>数量（残高）</th><th>基準価額</th><th>資産評価額</th><th>取得価額累計</th></tr><tr><td>broken</td><td>100円</td><td>100円</td><td>90円</td></tr><tr><th>解約価額</th><th>解約時評価額</th><th>損益</th></tr><tr><td>100円</td><td>100円</td><td>10円</td></tr><tr><th>基準日</th><th>資産比率</th></tr><tr><td>2026/09/11</td><td>100％</td></tr></table>`
	if _, err := ParseHoldingsHTML(fixture); err == nil {
		t.Fatal("malformed numeric should fail")
	}
}

func TestParseHoldingsHTMLPreservesStarDecimalUnitPrice(t *testing.T) {
	const fixture = `<div class="infoHdWrap"><dl><dt>商品名</dt><dd>架空債券</dd><dt>商品コード</dt><dd>07775</dd><dt>商品分類</dt><dd>国内投信</dd></dl></div><table><tr><th>数量（残高）</th><th>基準価額</th><th>資産評価額</th><th>取得価額累計</th></tr><tr><td>2</td><td>*1.25</td><td>100円</td><td>90円</td></tr><tr><th>解約価額</th><th>解約時評価額</th><th>損益</th></tr><tr><td>*1.20</td><td>100円</td><td>10円</td></tr><tr><th>基準日</th><th>資産比率</th></tr><tr><td>2026/09/11</td><td>100％</td></tr></table>`
	h, err := ParseHoldingsHTML(fixture)
	if err != nil {
		t.Fatal(err)
	}
	if h[0].UnitPrice != 1.25 || h[0].RedemptionUnitPrice != 1.2 || h[0].UnitPriceRaw != "*1.25" {
		t.Fatalf("prices=%+v", h[0])
	}
}

func TestParseHoldingsHTMLRejectsIncompleteSecondProduct(t *testing.T) {
	const first = `<div class="infoHdWrap"><dl><dt>商品名</dt><dd>架空一</dd><dt>商品コード</dt><dd>07771</dd><dt>商品分類</dt><dd>国内投信</dd></dl></div><table><tr><th>数量（残高）</th><th>基準価額</th><th>資産評価額</th><th>取得価額累計</th></tr><tr><td>1</td><td>100円</td><td>100円</td><td>90円</td></tr><tr><th>解約価額</th><th>解約時評価額</th><th>損益</th></tr><tr><td>100円</td><td>100円</td><td>10円</td></tr><tr><th>基準日</th><th>資産比率</th></tr><tr><td>2026/09/11</td><td>100％</td></tr></table>`
	second := `<div class="infoHdWrap"><dl><dt>商品名</dt><dd>架空二</dd><dt>商品コード</dt><dd>07772</dd><dt>商品分類</dt><dd>国内投信</dd></dl></div>`
	if _, err := ParseHoldingsHTML(first + second); err == nil {
		t.Fatal("incomplete second product should fail")
	}
}

func TestFetchAssetsUsesDisplayedTotalsAndWarnsOnDifference(t *testing.T) {
	const fixture = `<div class="infoHdWrap"><dl><dt>商品名</dt><dd>架空</dd><dt>商品コード</dt><dd>07776</dd><dt>商品分類</dt><dd>国内投信</dd></dl></div><table><tr><th>数量（残高）</th><th>基準価額</th><th>資産評価額</th><th>取得価額累計</th></tr><tr><td>1</td><td>100円</td><td>100円</td><td>90円</td></tr><tr><th>解約価額</th><th>解約時評価額</th><th>損益</th></tr><tr><td>100円</td><td>100円</td><td>10円</td></tr><tr><th>基準日</th><th>資産比率</th></tr><tr><td>2026/09/11</td><td>100％</td></tr></table><table><tr><th>資産評価額合計</th><th>取得価額累計合計</th><th>損益合計</th></tr><tr><td>101円</td><td>91円</td><td>11円</td></tr></table>`
	var logs bytes.Buffer
	f := &fakeSession{body: fixture}
	o := testOptions()
	o.Logger = slog.New(slog.NewTextHandler(&logs, nil))
	a, err := FetchAssets(context.Background(), f, o)
	if err != nil {
		t.Fatal(err)
	}
	if a.GrandTotalJPY != 101 || a.TotalCostJPY != 91 || a.PnLJPY != 11 {
		t.Fatalf("totals=%+v", a)
	}
	if !strings.Contains(logs.String(), "differ") {
		t.Fatal("expected reconciliation warning")
	}
}
