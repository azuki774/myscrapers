package main

import (
	"bytes"
	"context"
	"errors"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/azuki774/myscrapers/myscraper/internal/sbi"
)

func TestSBIMaintenanceDoesNotWriteAssets(t *testing.T) {
	for _, failedPage := range []int{2, 4} {
		for _, target := range []string{"stdout", "new-file", "existing-file"} {
			t.Run([]string{"", "", "foreign-assets", "", "foreign-summary"}[failedPage]+"/"+target, func(t *testing.T) {
				const notice = "現在、システムメンテナンスのため、当該サービスを停止させていただいております。"
				// Hand-written synthetic pages; no live account data.
				const totals = " 合計 評価額 含み損益 含み損益（％） 前日比 前日比（％） 0 0 0 0 0 "
				sess := &maintenanceSession{bodies: []string{
					"投資信託(金額/特定預り)" + totals + "投資信託(金額/旧つみたてNISA預り)" + totals,
					notice,
					"ダミー銘柄A DUMMYNASDAQ 100 USD 100 円 1 (0) 100 USD 100 円 100 USD 100 円 100 USD 100 円 0 USD 0 円 現買 現売 積立",
					"現金残高等\n0",
					notice,
				}, index: -1}
				sess.bodies[failedPage] = notice
				var stdout, logs bytes.Buffer
				runner := sbiRunner{
					stdout:     &stdout,
					logger:     slog.New(slog.NewJSONHandler(&logs, nil)),
					newSession: func(context.Context, bool) (sbi.Session, error) { return sess, nil },
				}
				var outputPath string
				if target != "stdout" {
					outputPath = filepath.Join(t.TempDir(), "assets.json")
				}
				if target == "existing-file" {
					if err := os.WriteFile(outputPath, []byte("existing-output"), 0600); err != nil {
						t.Fatal(err)
					}
				}
				store := &maintenanceStore{}
				err := runner.RunAssets(context.Background(), sbi.FetchOptions{OutputPath: outputPath, S3Upload: true, S3Client: store})
				if !errors.Is(err, sbi.ErrMaintenance) {
					t.Fatalf("want maintenance, got %v", err)
				}
				if !sess.closed || sess.index != failedPage {
					t.Fatal("browser not closed or navigation continued")
				}
				if stdout.Len() != 0 || store.uploaded {
					t.Fatal("failed fetch wrote asset data")
				}
				if target == "existing-file" {
					content, err := os.ReadFile(outputPath)
					if err != nil || string(content) != "existing-output" {
						t.Fatal("existing output changed")
					}
				} else if target == "new-file" {
					if _, err := os.Stat(outputPath); !errors.Is(err, os.ErrNotExist) {
						t.Fatal("output file created")
					}
				}
				for _, stage := range []string{"output", "s3_upload", "resolve_figi"} {
					if strings.Contains(logs.String(), `"stage":"`+stage+`"`) {
						t.Fatalf("unexpected stage %s", stage)
					}
				}
			})
		}
	}
}

type maintenanceSession struct {
	bodies []string
	index  int
	closed bool
}

func (s *maintenanceSession) LoginWithPasskey(context.Context, *sbi.PasskeyFile) error { return nil }
func (s *maintenanceSession) Goto(context.Context, string) error                       { s.index++; return nil }
func (s *maintenanceSession) BodyText(context.Context) (string, error)                 { return s.bodies[s.index], nil }
func (s *maintenanceSession) BodyHTML(context.Context) (string, error)                 { return "", nil }
func (s *maintenanceSession) Wait(context.Context, time.Duration) error                { return nil }
func (s *maintenanceSession) Close() error                                             { s.closed = true; return nil }

type maintenanceStore struct{ uploaded bool }

func (s *maintenanceStore) PutJSON(context.Context, string, io.Reader) error {
	s.uploaded = true
	return nil
}
func (s *maintenanceStore) KeyForTime(time.Time) string { return "dummy.json" }
