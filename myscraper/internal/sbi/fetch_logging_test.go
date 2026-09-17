package sbi

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"testing"
	"time"
)

func TestFetchAssetsLogsEachPageAndFIGIResolution(t *testing.T) {
	sess := &fakeSession{bodies: map[string]string{
		portfolioURL:       portfolioFixture,
		nisaPortfolioURL:   nisaFixture,
		foreignAssetsURL:   foreignHoldingsFixture,
		domesticSummaryURL: cashFixture,
		foreignSummaryURL:  foreignCashFixture,
	}, htmls: map[string]string{portfolioURL: portfolioHTMLFixture}}
	var output bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&output, nil))
	if _, err := FetchAssets(context.Background(), sess, time.Now(), staticFIGIResolver{}, logger); err != nil {
		t.Fatalf("FetchAssets: %v", err)
	}

	var records []map[string]any
	decoder := json.NewDecoder(&output)
	for {
		var record map[string]any
		err := decoder.Decode(&record)
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatalf("decode log: %v", err)
		}
		records = append(records, record)
	}
	wantPages := map[string]bool{"portfolio": false, "nisa_portfolio": false, "foreign_assets": false, "domestic_summary": false, "foreign_summary": false}
	for _, record := range records {
		if record["scraper"] != "sbi" {
			t.Errorf("scraper = %v, want sbi", record["scraper"])
		}
		if record["stage"] == "page" && record["event"] == "completed" {
			page, ok := record["page"].(string)
			if !ok {
				t.Fatalf("completed page record has no page: %#v", record)
			}
			if _, ok := wantPages[page]; !ok {
				t.Errorf("unexpected page %q", page)
			} else {
				wantPages[page] = true
			}
		}
	}
	for page, seen := range wantPages {
		if !seen {
			t.Errorf("page %q has no completed record", page)
		}
	}
	if !hasCompletedRecord(records, "resolve_figi") {
		t.Error("resolve_figi has no completed record")
	}
}

func hasCompletedRecord(records []map[string]any, stage string) bool {
	for _, record := range records {
		if record["stage"] == stage && record["event"] == "completed" {
			return true
		}
	}
	return false
}
