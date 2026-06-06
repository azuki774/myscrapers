package moneyforward

import (
	"strings"
	"testing"
)

const cfHTML = `
<html><body>
<table id="cf-detail-table">
  <tr><th>x</th></tr>
  <tr><td></td><td>12/09(月)</td><td>物販</td><td>-110</td><td>モバイルSuica</td><td>未分類</td><td>未分類</td><td></td><td></td><td></td></tr>
  <tr><td></td><td>12/10(火)</td><td>コンビニ</td><td>-580</td><td>三井住友</td><td>食費</td><td>外食</td><td>メモ</td><td></td><td></td></tr>
  <tr><td></td><td></td><td></td><td></td><td></td><td></td><td></td><td></td><td></td><td></td></tr>
</table>
</body></html>
`

const assetHTML = `
<html><body>
<table class="table table-bordered">
  <thead>
    <tr><th>月</th><th>資産合計</th></tr>
  </thead>
  <tbody>
    <tr><td>2024-12</td><td>1,234,567円</td><td><a href="/d/1">詳細</a></td></tr>
    <tr><td>2024-11</td><td>1,200,000円</td><td><a href="/d/2">詳細</a></td></tr>
  </tbody>
</table>
</body></html>
`

func TestExtractCFTableSkipsEmptyRows(t *testing.T) {
	rows, err := ExtractCFTable(cfHTML)
	if err != nil {
		t.Fatalf("ExtractCFTable() error = %v", err)
	}
	if len(rows) != 2 {
		t.Fatalf("len(rows) = %d, want 2; rows = %+v", len(rows), rows)
	}
	if !strings.HasPrefix(rows[0][1], "12/09") {
		t.Fatalf("rows[0][1] = %q", rows[0][1])
	}
}

func TestExtractAssetHistoryTableStripsYenAndKeepsShosaiLink(t *testing.T) {
	rows, err := ExtractAssetHistoryTable(assetHTML)
	if err != nil {
		t.Fatalf("ExtractAssetHistoryTable() error = %v", err)
	}
	if len(rows) != 3 {
		t.Fatalf("len(rows) = %d, want 3 (header + 2 body); rows = %+v", len(rows), rows)
	}
	if rows[0][0] != "月" || rows[0][1] != "資産合計" {
		t.Fatalf("rows[0] = %+v", rows[0])
	}
	if rows[1][1] != "1,234,567" {
		t.Fatalf("rows[1][1] = %q, want %q (yen stripped)", rows[1][1], "1,234,567")
	}
	if rows[1][2] != "詳細" {
		t.Fatalf("rows[1][2] = %q, want 詳細", rows[1][2])
	}
}
