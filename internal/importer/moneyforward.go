package moneyforward

import (
	"context"
	"fmt"
	"log/slog"
	"myscrapers/internal/csv"
	"strings"

	"github.com/go-rod/rod"
)

const cfFieldSize = 10

func validateCF(header []string, bodies [][]string) error {
	if len(header) != cfFieldSize {
		return fmt.Errorf("invalid field size: header")
	}
	for i, b := range bodies {
		if len(b) != cfFieldSize {
			return fmt.Errorf("invalid field size: bodies #%d", i+1)
		}
	}
	return nil
}

// cfPage は /cf のページを rod で取得したもの
func getHeader(ctx context.Context, cfPage *rod.Page) (header []string, err error) {
	cfDetailTable, err := cfPage.Element(`[id=cf-detail-table]`)
	if err != nil {
		slog.Error("failed to get cf-detail-table")
		return []string{}, err
	}

	ths, err := cfDetailTable.Elements("th")
	if err != nil {
		slog.Error("failed to get cfDetailTable")
		return []string{}, err
	}

	for _, th := range ths {
		// セレクターの選択肢のテキストを消す
		txt := strings.Split(th.MustText(), " ")[0]
		// 改行を消す
		txt = strings.ReplaceAll(txt, "\n", "")
		// 無駄な空白を消す
		txt = strings.ReplaceAll(txt, " ", "")
		header = append(header, txt)
	}

	return header, nil
}

// cfPage は /cf のページを rod で取得したもの
func getBody(ctx context.Context, cfPage *rod.Page) (bodies [][]string, err error) {
	cfDetailTable, err := cfPage.Element(`[id=cf-detail-table]`)
	if err != nil {
		slog.Error("failed to get cf-detail-table")
		return [][]string{}, err
	}

	recordRows, err := cfDetailTable.Elements(`[class="transaction_list js-cf-edit-container target-active"`) // 1行ごとのrecordsのフィールドを特定する
	if err != nil {
		slog.Error("failed to get recordRows")
		return [][]string{}, err
	}

	for _, recordRow := range recordRows {
		var row []string
		spans := recordRow.MustElements("td") // 1行ごとのレコードから各セルを抽出
		for _, span := range spans {
			// セレクターの選択肢のテキストを消す
			txt := strings.Split(span.MustText(), " ")[0]
			// 改行を消す
			txt = strings.ReplaceAll(txt, "\n", "")
			// 無駄な空白を消さない
			row = append(row, txt)
		}
		bodies = append(bodies, row)
	}

	return bodies, nil
}

func ImportStart(ctx context.Context, filePath string, page *rod.Page) (err error) {
	var header []string
	var bodies [][]string

	header, err = getHeader(ctx, page)
	if err != nil {
		slog.Error("failed to get header")
		return err
	}
	slog.Info("get CSV header")

	bodies, err = getBody(ctx, page)
	if err != nil {
		slog.Error("failed to get bodies")
		return err
	}
	slog.Info("get CSV body")

	// validation
	if err := validateCF(header, bodies); err != nil {
		return err
	}

	// csv書き込み
	if err := csv.WriteFile(filePath, header, bodies); err != nil {
		slog.Error("failed to output csv")
		return err
	}
	slog.Info("output csv complete", "outputPath", filePath)
	return nil
}
