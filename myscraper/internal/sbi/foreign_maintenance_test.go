package sbi

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"strings"
	"testing"
	"time"

	"github.com/azuki774/myscrapers/myscraper/internal/logging"
)

const foreignMaintenanceNotice = "現在、システムメンテナンスのため、当該サービスを停止させていただいております。"

func TestMaintenanceStopNotice(t *testing.T) {
	for _, tc := range []struct {
		name string
		body string
		want bool
	}{
		{"stop notice", foreignMaintenanceNotice, true},
		{"whitespace", "現在、\nシステム\tメンテナンス\u3000のため、停止中です。", true},
		{"notice link", "定期・臨時システムメンテナンスのお知らせ", false},
		{"legacy notice link", "臨時メンテナンスのお知らせ", false},
		{"healthy page with notice", foreignCashFixture + "\nメンテナンスのお知らせ", false},
		{"unknown notice", "サービスを一時停止しています。", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := isMaintenancePage(tc.body); got != tc.want {
				t.Fatalf("maintenance = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestForeignPageFailureStopsFetch(t *testing.T) {
	for _, page := range []struct{ name, url string }{
		{"foreign_assets", foreignAssetsURL},
		{"foreign_summary", foreignSummaryURL},
	} {
		for _, tc := range []struct {
			name string
			body string
			want error
		}{
			{"maintenance", foreignMaintenanceNotice, ErrMaintenance},
			{"unknown", "画面を表示できません。", ErrUnexpectedPage},
			{"empty", "", ErrUnexpectedPage},
			{"notice link only", "メンテナンスのお知らせ", ErrUnexpectedPage},
		} {
			t.Run(page.name+"/"+tc.name, func(t *testing.T) {
				sess := healthyForeignSession()
				sess.bodies[page.url] = tc.body
				var logs bytes.Buffer
				assets, err := FetchAssets(context.Background(), sess, time.Now(), failingFIGIResolver{}, slog.New(slog.NewJSONHandler(&logs, nil)))
				if assets != nil || !errors.Is(err, tc.want) {
					t.Fatalf("want nil assets and %v; got error %v", tc.want, err)
				}
				stage, name, ok := logging.Context(err)
				if !ok || stage != "page" || name != page.name || sess.last != page.url {
					t.Fatal("failure lost page context or continued navigation")
				}
				for _, line := range strings.Split(logs.String(), "\n") {
					if strings.Contains(line, `"page":"`+page.name+`"`) && strings.Contains(line, `"event":"completed"`) {
						t.Fatal("failed page emitted completed")
					}
				}
				if strings.Contains(logs.String(), "resolve_figi") {
					t.Fatal("FIGI resolution started after page failure")
				}
			})
		}
	}
}

func TestForeignNoticeLinksDoNotStopFetch(t *testing.T) {
	sess := healthyForeignSession()
	for _, url := range []string{foreignAssetsURL, foreignSummaryURL, nisaPortfolioURL} {
		sess.bodies[url] += "\n定期・臨時システムメンテナンスのお知らせ"
	}
	assets, err := FetchAssets(context.Background(), sess, time.Now(), staticFIGIResolver{}, nil)
	if err != nil || assets == nil || assets.Status != StatusOK {
		t.Fatalf("healthy pages with notice links failed: %v", err)
	}
}

func healthyForeignSession() *fakeSession {
	return &fakeSession{bodies: map[string]string{
		portfolioURL:       portfolioFixture,
		nisaPortfolioURL:   nisaFixture,
		foreignAssetsURL:   foreignHoldingsFixture,
		domesticSummaryURL: cashFixture,
		foreignSummaryURL:  foreignCashFixture,
	}, htmls: map[string]string{portfolioURL: portfolioHTMLFixture}}
}
