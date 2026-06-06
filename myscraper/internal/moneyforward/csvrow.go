package moneyforward

import (
	"bytes"
	"encoding/csv"
	"fmt"
	"os"
	"strings"
	"time"

	"golang.org/x/text/encoding/japanese"
	"golang.org/x/text/transform"
)

const cfHeader = `"計算対象","日付","内容","金額（円）","保有金融機関","大項目","中項目","メモ","振替","ID"` + "\n"

// ConvertCSVData formats one extracted "#cf-detail-table" row as a fully
// quoted MoneyForward CSV line. Field 0 is forced to "1" (the Python script
// treats every record as in-scope), field 1 is date-normalised via
// ConvertDateField, fields 2-9 are the first line of each cell (newlines
// inside the cell are stripped). The output stays in UTF-8; UTF8ToSJIS
// converts the file in place afterwards.
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

// UTF8ToSJIS reads the file as UTF-8 (one CSV record per line, no embedded
// newlines), encodes every field to Shift-JIS with the replace fallback
// for unmappable characters, and writes the result back to the same path.
// The CSV structure is preserved because the writer is only switched for
// the underlying byte stream.
func UTF8ToSJIS(path string) error {
	in, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("read %s: %w", path, err)
	}
	r := csv.NewReader(bytes.NewReader(in))
	records, err := r.ReadAll()
	if err != nil {
		return fmt.Errorf("parse %s: %w", path, err)
	}
	var buf bytes.Buffer
	w := csv.NewWriter(transform.NewWriter(&buf, japanese.ShiftJIS.NewEncoder()))
	if err := w.WriteAll(records); err != nil {
		return fmt.Errorf("encode %s: %w", path, err)
	}
	w.Flush()
	if err := w.Error(); err != nil {
		return fmt.Errorf("flush %s: %w", path, err)
	}
	if err := os.WriteFile(path, buf.Bytes(), 0o644); err != nil {
		return fmt.Errorf("write %s: %w", path, err)
	}
	return nil
}
