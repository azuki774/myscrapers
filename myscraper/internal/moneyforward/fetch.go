package moneyforward

import (
	"context"
	"encoding/csv"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"time"
)

const (
	cfURL               = "https://moneyforward.com/cf"
	bsHistoryURL        = "https://moneyforward.com/bs/history"
	cfFilename          = "cf.csv"
	cfLastFilename      = "cf_lastmonth.csv"
	assetFilename       = "asset_history.csv"
	cfNowDebugFilename  = "cf_now_debug.html"
	cfLastDebugFilename = "cf_lastmonth_debug.html"
	lastMonthXPath      = "/html/body/div[1]/div[2]/div/div/section/div[2]/button[1]"
	cookiePrimSleep     = 10 * time.Second
	postClickSleep      = 5 * time.Second
)

// FetchOptions collects the inputs the Fetch orchestrator needs from the
// caller. Session and Cookies are required; OutputDir defaults to /data
// in the CLI layer; Now defaults to time.Now() in the CLI layer;
// Uploader is optional and triggers the S3 upload step when set.
type FetchOptions struct {
	Session   Session
	Cookies   []Cookie
	OutputDir string
	Now       time.Time
	Uploader  Uploader
	Logger    *slog.Logger
}

// Fetch orchestrates the "moneyforward --fetch" subcommand: prime the
// cookie domain, inject cookies, scrape CF (this month + last month) and
// the asset history page, write the three CSVs, convert them in place to
// Shift-JIS, and (if Uploader is set in the wrapper) upload them. The
// returned error wraps the first failure; on success Session.Close is
// always called.
func Fetch(ctx context.Context, opts FetchOptions) error {
	if opts.Session == nil {
		return fmt.Errorf("session is required")
	}
	if opts.OutputDir == "" {
		opts.OutputDir = "/data"
	}
	if opts.Now.IsZero() {
		opts.Now = time.Now()
	}
	logger := opts.Logger
	if logger == nil {
		logger = slog.New(slog.NewJSONHandler(io.Discard, nil))
	}
	defer opts.Session.Close()

	if err := opts.Session.Goto(ctx, cfURL); err != nil {
		return fmt.Errorf("goto cf: %w", err)
	}
	if err := opts.Session.Wait(ctx, cookiePrimSleep); err != nil {
		return fmt.Errorf("wait after cf: %w", err)
	}
	if err := opts.Session.AddCookies(ctx, opts.Cookies); err != nil {
		return fmt.Errorf("add cookies: %w", err)
	}
	if err := opts.Session.Goto(ctx, cfURL); err != nil {
		return fmt.Errorf("re-goto cf: %w", err)
	}
	if err := opts.Session.Wait(ctx, postClickSleep); err != nil {
		return fmt.Errorf("wait after re-goto cf: %w", err)
	}

	nowHTML, err := opts.Session.Content(ctx)
	if err != nil {
		return fmt.Errorf("read cf now: %w", err)
	}
	nowRows, err := ExtractCFTable(nowHTML)
	if err != nil {
		debugPath, dumpErr := writeDebugHTML(opts.OutputDir, cfNowDebugFilename, nowHTML)
		if dumpErr != nil {
			return fmt.Errorf("parse cf now: %w; also failed to write debug html: %v", err, dumpErr)
		}
		return fmt.Errorf("parse cf now: %w; debug html: %s", err, debugPath)
	}
	if err := writeCF(opts, nowRows, false); err != nil {
		return err
	}
	logger.Info("fetched current month", "rows", len(nowRows))

	if err := opts.Session.ClickByXPath(ctx, lastMonthXPath); err != nil {
		return fmt.Errorf("click last month: %w", err)
	}
	if err := opts.Session.Wait(ctx, postClickSleep); err != nil {
		return fmt.Errorf("wait after last month click: %w", err)
	}
	if err := opts.Session.Goto(ctx, cfURL); err != nil {
		return fmt.Errorf("re-goto cf for last month: %w", err)
	}
	if err := opts.Session.Wait(ctx, postClickSleep); err != nil {
		return fmt.Errorf("wait after re-goto cf for last month: %w", err)
	}
	lastHTML, err := opts.Session.Content(ctx)
	if err != nil {
		return fmt.Errorf("read cf lastmonth: %w", err)
	}
	lastRows, err := ExtractCFTable(lastHTML)
	if err != nil {
		debugPath, dumpErr := writeDebugHTML(opts.OutputDir, cfLastDebugFilename, lastHTML)
		if dumpErr != nil {
			return fmt.Errorf("parse cf lastmonth: %w; also failed to write debug html: %v", err, dumpErr)
		}
		return fmt.Errorf("parse cf lastmonth: %w; debug html: %s", err, debugPath)
	}
	if err := writeCF(opts, lastRows, true); err != nil {
		return err
	}
	logger.Info("fetched last month", "rows", len(lastRows))

	if err := opts.Session.Goto(ctx, bsHistoryURL); err != nil {
		return fmt.Errorf("goto bs/history: %w", err)
	}
	if err := opts.Session.Wait(ctx, postClickSleep); err != nil {
		return fmt.Errorf("wait after bs/history: %w", err)
	}
	bsHTML, err := opts.Session.Content(ctx)
	if err != nil {
		return fmt.Errorf("read bs/history: %w", err)
	}
	assetRows, err := ExtractAssetHistoryTable(bsHTML)
	if err != nil {
		return fmt.Errorf("parse bs/history: %w", err)
	}
	if err := writeAssetHistory(opts.OutputDir, assetRows); err != nil {
		return err
	}
	logger.Info("fetched asset history", "rows", len(assetRows))

	for _, name := range []string{cfFilename, cfLastFilename, assetFilename} {
		if err := UTF8ToSJIS(filepath.Join(opts.OutputDir, name)); err != nil {
			return fmt.Errorf("convert %s: %w", name, err)
		}
	}
	if opts.Uploader != nil {
		for _, name := range []string{cfFilename, cfLastFilename, assetFilename} {
			path := filepath.Join(opts.OutputDir, name)
			if err := opts.Uploader.Upload(ctx, path); err != nil {
				return fmt.Errorf("upload %s: %w", name, err)
			}
		}
		logger.Info("uploaded 3 csv files")
	}
	return nil
}

func writeCF(opts FetchOptions, rows [][]string, lastmonth bool) error {
	out := make([]string, 0, len(rows))
	for _, row := range rows {
		out = append(out, ConvertCSVData(row, lastmonth, opts.Now))
	}
	name := cfFilename
	if lastmonth {
		name = cfLastFilename
	}
	if err := WriteCFCSV(filepath.Join(opts.OutputDir, name), out); err != nil {
		return fmt.Errorf("write %s: %w", name, err)
	}
	return nil
}

func writeAssetHistory(dir string, rows [][]string) error {
	path := filepath.Join(dir, assetFilename)
	f, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("create %s: %w", path, err)
	}
	defer f.Close()
	w := csv.NewWriter(f)
	if err := w.WriteAll(rows); err != nil {
		return fmt.Errorf("write %s: %w", path, err)
	}
	w.Flush()
	return w.Error()
}

func writeDebugHTML(dir, name, body string) (string, error) {
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
		return path, fmt.Errorf("write %s: %w", path, err)
	}
	return path, nil
}
