package moneyforward

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"testing"
	"time"

	"golang.org/x/text/encoding/japanese"
	"golang.org/x/text/transform"
)

func TestConvertCSVData(t *testing.T) {
	now := time.Date(2024, 1, 23, 0, 0, 0, 0, time.UTC)
	row := []string{"", "12/09(月)", "物販", "-110", "モバイルSuica", "未分類", "未分類", "", "", ""}
	got := ConvertCSVData(row, false, now)
	want := `"1","2024/12/09","物販","-110","モバイルSuica","未分類","未分類","","",""`
	if got != want {
		t.Fatalf("ConvertCSVData() = %q, want %q", got, want)
	}
}

func TestWriteCFCSVAndUTF8ToSJIS(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "cf.csv")
	rows := []string{
		`"1","2024/12/09","物販","-110","モバイルSuica","未分類","未分類","","",""`,
		`"1","2024/12/10","コンビニ","-580","三井住友","食費","外食","メモ","",""`,
	}
	if err := WriteCFCSV(path, rows); err != nil {
		t.Fatalf("WriteCFCSV() error = %v", err)
	}
	if err := UTF8ToSJIS(path); err != nil {
		t.Fatalf("UTF8ToSJIS() error = %v", err)
	}
	// Read back the file and decode Shift-JIS to UTF-8 in-memory. This
	// verifies the round-trip: WriteCFCSV writes UTF-8, UTF8ToSJIS converts
	// to Shift-JIS, and decoding the file recovers the original Japanese.
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}
	decoded, err := io.ReadAll(transform.NewReader(bytes.NewReader(raw), japanese.ShiftJIS.NewDecoder()))
	if err != nil {
		t.Fatalf("SJIS decode() error = %v", err)
	}
	text := string(decoded)
	for _, frag := range []string{`計算対象`, `モバイルSuica`, `三井住友`, `2024/12/09`, `2024/12/10`} {
		if !containsBytes([]byte(text), []byte(frag)) {
			t.Fatalf("expected fragment %q in decoded text:\n%s", frag, text)
		}
	}
}

func containsBytes(haystack []byte, needle []byte) bool {
	if len(needle) == 0 {
		return true
	}
	for i := 0; i+len(needle) <= len(haystack); i++ {
		match := true
		for j := 0; j < len(needle); j++ {
			if haystack[i+j] != needle[j] {
				match = false
				break
			}
		}
		if match {
			return true
		}
	}
	return false
}
