package sbi

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"
	"time"
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

func TestOpenFIGIResolverBatchesAndDeduplicates(t *testing.T) {
	var batches [][]FIGILookup
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var requests []openFIGIRequest
		if err := json.NewDecoder(r.Body).Decode(&requests); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		batch := make([]FIGILookup, len(requests))
		response := make([]openFIGIResult, len(requests))
		for i, request := range requests {
			batch[i] = FIGILookup{Ticker: request.IDValue, ExchCode: request.ExchCode}
			response[i].Data = []openFIGIData{{Ticker: request.IDValue, ExchCode: request.ExchCode, CompositeFIGI: "FIGI-" + request.IDValue}}
		}
		batches = append(batches, batch)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()
	client := NewOpenFIGIClientForEndpoint(server.URL, server.Client())
	client.sleep = func(context.Context, time.Duration) error { return nil }
	resolver := NewOpenFIGIResolver(client)
	lookups := make([]FIGILookup, 0, 21)
	for i := 0; i < 21; i++ {
		lookups = append(lookups, FIGILookup{Ticker: "T" + string(rune('A'+i)), ExchCode: "US"})
	}
	lookups = append(lookups, lookups[0])
	resolved, err := resolver.Resolve(context.Background(), lookups)
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if len(batches) != 3 || len(batches[0]) != 10 || len(batches[1]) != 10 || len(batches[2]) != 1 {
		t.Fatalf("batches = %#v", batches)
	}
	if len(resolved) != 21 || resolved[lookups[0]] != "FIGI-TA" {
		t.Fatalf("resolved = %#v", resolved)
	}
}

func TestOpenFIGIClientRetries429AndHonorsRetryAfter(t *testing.T) {
	attempts := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts++
		if attempts == 1 {
			w.Header().Set("Retry-After", "0")
			w.WriteHeader(http.StatusTooManyRequests)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		ioResponse := `[ {"data":[{"ticker":"AMD","exchCode":"US","compositeFIGI":"FIGI-AMD"}]} ]`
		_, _ = w.Write([]byte(ioResponse))
	}))
	defer server.Close()
	client := NewOpenFIGIClientForEndpoint(server.URL, server.Client())
	var delays []time.Duration
	client.sleep = func(_ context.Context, delay time.Duration) error { delays = append(delays, delay); return nil }
	resolved, err := NewOpenFIGIResolver(client).Resolve(context.Background(), []FIGILookup{{Ticker: "AMD", ExchCode: "US"}})
	if err != nil || resolved[FIGILookup{Ticker: "AMD", ExchCode: "US"}] != "FIGI-AMD" {
		t.Fatalf("resolved=%#v err=%v", resolved, err)
	}
	if attempts != 2 || !reflect.DeepEqual(delays, []time.Duration{0}) {
		t.Fatalf("attempts=%d delays=%v", attempts, delays)
	}
}

func TestOpenFIGIResolverRejectsInvalidResultsAndSkipsEmptyCall(t *testing.T) {
	called := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { called++ }))
	defer server.Close()
	client := NewOpenFIGIClientForEndpoint(server.URL, server.Client())
	if got, err := NewOpenFIGIResolver(client).Resolve(context.Background(), nil); err != nil || len(got) != 0 || called != 0 {
		t.Fatalf("empty Resolve got=%#v err=%v calls=%d", got, err, called)
	}
	server2 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`[{"data":[]}]`))
	}))
	defer server2.Close()
	client = NewOpenFIGIClientForEndpoint(server2.URL, server2.Client())
	if _, err := NewOpenFIGIResolver(client).Resolve(context.Background(), []FIGILookup{{Ticker: "BAD", ExchCode: "JP"}}); err == nil {
		t.Fatal("invalid empty result succeeded")
	}
}

func TestOpenFIGIResolverRejectsMalformedResponses(t *testing.T) {
	tests := []struct {
		name    string
		status  int
		body    string
		wantErr string
	}{
		{name: "multiple results", body: `[{"data":[{"ticker":"AMD","exchCode":"US","compositeFIGI":"A"},{"ticker":"AMD","exchCode":"US","compositeFIGI":"B"}]}]`, wantErr: "returned 2 results"},
		{name: "mismatch", body: `[{"data":[{"ticker":"NVDA","exchCode":"US","compositeFIGI":"A"}]}]`, wantErr: "result mismatch"},
		{name: "empty composite", body: `[{"data":[{"ticker":"AMD","exchCode":"US"}]}]`, wantErr: "result mismatch"},
		{name: "response count", body: `[]`, wantErr: "response count"},
		{name: "warning", body: `[{"warning":"rate limited"}]`, wantErr: "warning: rate limited"},
		{name: "error", body: `[{"error":"bad ticker"}]`, wantErr: "error: bad ticker"},
		{name: "malformed json", body: `{`, wantErr: "decode OpenFIGI response"},
		{name: "non retryable HTTP", status: http.StatusBadRequest, body: `bad request`, wantErr: "HTTP status 400"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if tt.status != 0 {
					w.WriteHeader(tt.status)
				}
				_, _ = w.Write([]byte(tt.body))
			}))
			defer server.Close()
			client := NewOpenFIGIClientForEndpoint(server.URL, server.Client())
			client.sleep = func(context.Context, time.Duration) error { return nil }
			_, err := NewOpenFIGIResolver(client).Resolve(context.Background(), []FIGILookup{{Ticker: "AMD", ExchCode: "US"}})
			if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
				t.Fatalf("error = %v, want substring %q", err, tt.wantErr)
			}
		})
	}
}

func TestOpenFIGIClientRetriesAndExhausts5xx(t *testing.T) {
	attempts := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts++
		w.WriteHeader(http.StatusBadGateway)
	}))
	defer server.Close()
	client := NewOpenFIGIClientForEndpoint(server.URL, server.Client())
	var delays []time.Duration
	client.sleep = func(_ context.Context, delay time.Duration) error { delays = append(delays, delay); return nil }
	_, err := NewOpenFIGIResolver(client).Resolve(context.Background(), []FIGILookup{{Ticker: "AMD", ExchCode: "US"}})
	if err == nil || !strings.Contains(err.Error(), "after 3 attempts") {
		t.Fatalf("error = %v", err)
	}
	if attempts != 3 || !reflect.DeepEqual(delays, []time.Duration{time.Second, 2 * time.Second}) {
		t.Fatalf("attempts=%d delays=%v", attempts, delays)
	}
}
