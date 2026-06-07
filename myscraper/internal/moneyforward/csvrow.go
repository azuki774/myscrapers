package moneyforward

import (
	"fmt"
	"os"
	"strings"
	"time"
)

const cfHeader = `"計算対象","日付","内容","金額（円）","保有金融機関","大項目","中項目","メモ","振替","ID"` + "\n"

// ConvertCSVData formats one extracted "#cf-detail-table" row as a fully
// quoted MoneyForward CSV line. Field 0 is forced to "1" (the Python script
// treats every record as in-scope), field 1 is date-normalised via
// ConvertDateField, fields 2-9 are the first line of each cell (newlines
// inside the cell are stripped).
func ConvertCSVData(row []string, lastmonth bool, now time.Time) string {
	get := func(i int) string {
		if i >= len(row) {
			return ""
		}
		return strings.SplitN(row[i], "\n", 2)[0]
	}
	return fmt.Sprintf(
		`"%s","%s","%s","%s","%s","%s","%s","%s","%s","%s"`,
		"1",
		ConvertDateField(row[1], lastmonth, now),
		get(2), get(3), get(4), get(5), get(6), get(7), get(8), get(9),
	)
}

// WriteCFCSV writes the hard-coded MoneyForward header followed by the
// pre-formatted CSV rows. It uses a buffered write so callers do not have
// to think about flushing.
func WriteCFCSV(path string, rows []string) error {
	if err := os.WriteFile(path, []byte(cfHeader), 0o644); err != nil {
		return fmt.Errorf("write cf header: %w", err)
	}
	f, err := os.OpenFile(path, os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		return fmt.Errorf("open cf file for append: %w", err)
	}
	defer f.Close()
	for _, r := range rows {
		if _, err := f.WriteString(r + "\n"); err != nil {
			return fmt.Errorf("append cf row: %w", err)
		}
	}
	return nil
}
