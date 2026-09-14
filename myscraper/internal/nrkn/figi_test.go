package nrkn

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/azuki774/myscrapers/myscraper/internal/figi"
)

func TestFundLookupVerifiedNames(t *testing.T) {
	for _, tc := range []struct{ name, ticker string }{
		{"ＤＣニッセイ国内株式インデックス", "29316149"},
		{" DC ニッセイ国内株式インデックス ", "29316149"},
		{"野村外株インデックスファンド・ＭＳＣＩ－ＫＯＫＵＳＡＩ・ＤＣ", "01312022"},
		{"マイバランス７０（確定拠出年金向け）・野村", "01314027"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got, err := fundLookup(Holding{ProductCode: "09999", Name: tc.name, Category: "国内投信"})
			if err != nil || got != (figi.FIGILookup{Ticker: tc.ticker, ExchCode: "JP"}) {
				t.Fatalf("lookup=%+v err=%v", got, err)
			}
		})
	}
}

func TestHoldingFIGIResolverMappingAndFailures(t *testing.T) {
	for _, scenario := range []string{"success", "unknown", "wrong-category", "similar-name", "duplicate", "missing", "ambiguous"} {
		t.Run(scenario, func(t *testing.T) {
			calls := 0
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
				calls++
				var jobs []struct {
					IDType   string `json:"idType"`
					IDValue  string `json:"idValue"`
					ExchCode string `json:"exchCode"`
				}
				if req.Method != http.MethodPost || req.URL.Path != "/v3/mapping" {
					t.Errorf("unexpected request %s %s", req.Method, req.URL.Path)
				}
				if err := json.NewDecoder(req.Body).Decode(&jobs); err != nil {
					t.Error(err)
					w.WriteHeader(http.StatusBadRequest)
					return
				}
				if len(jobs) != 1 || jobs[0].IDType != "TICKER" || jobs[0].IDValue != "29316149" || jobs[0].ExchCode != "JP" {
					t.Errorf("unexpected jobs %+v", jobs)
				}
				data := []map[string]string{{"ticker": "29316149", "exchCode": "JP", "compositeFIGI": "BBGTESTFUND"}}
				if scenario == "missing" {
					data[0]["compositeFIGI"] = ""
				}
				if scenario == "ambiguous" {
					data = append(data, data[0])
				}
				_ = json.NewEncoder(w).Encode([]any{map[string]any{"data": data}})
			}))
			defer server.Close()
			r := NewHoldingFIGIResolver(figi.NewOpenFIGIResolver(figi.NewOpenFIGIClientForEndpoint(server.URL+"/v3/mapping", server.Client())))
			h := Holding{ProductCode: "09999", Name: "ＤＣニッセイ国内株式インデックス", Category: "国内投信"}
			switch scenario {
			case "unknown":
				h.Name = "架空ファンド"
			case "wrong-category":
				h.Category = "定期預金"
			case "similar-name":
				h.Name += "（為替ヘッジあり）"
			}
			holdings := []Holding{h}
			if scenario == "duplicate" {
				holdings = append(holdings, h)
			}
			if scenario == "success" {
				other := h
				other.ProductCode = "08888"
				holdings = append(holdings, other)
			}
			got, err := r.Resolve(context.Background(), holdings)
			if scenario == "success" {
				if err != nil || len(got) != 2 || got["09999"] != "BBGTESTFUND" || got["08888"] != "BBGTESTFUND" {
					t.Fatalf("got=%v err=%v", got, err)
				}
			} else if err == nil || got != nil {
				t.Fatalf("expected no results on failure: got=%v err=%v", got, err)
			}
			wantCalls := 1
			if scenario == "unknown" || scenario == "wrong-category" || scenario == "similar-name" || scenario == "duplicate" {
				wantCalls = 0
			}
			if calls != wantCalls {
				t.Fatalf("calls=%d want %d", calls, wantCalls)
			}
		})
	}
}
