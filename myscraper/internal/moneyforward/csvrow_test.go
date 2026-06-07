package moneyforward

import (
	"os"
	"path/filepath"
	"testing"
	"time"
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

func TestWriteCFCSV(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "cf.csv")
	rows := []string{
		`"1","2024/12/09","物販","-110","モバイルSuica","未分類","未分類","","",""`,
		`"1","2024/12/10","コンビニ","-580","三井住友","食費","外食","メモ","",""`,
	}
	if err := WriteCFCSV(path, rows); err != nil {
		t.Fatalf("WriteCFCSV() error = %v", err)
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}
	text := string(raw)
	for _, frag := range []string{`計算対象`, `モバイルSuica`, `三井住友`, `2024/12/09`, `2024/12/10`} {
		if !containsBytes([]byte(text), []byte(frag)) {
			t.Fatalf("expected fragment %q in text:\n%s", frag, text)
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
