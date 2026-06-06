package moneyforward

import (
	"fmt"
	"strings"

	"golang.org/x/net/html"
)

// ExtractCFTable parses a MoneyForward /cf page snapshot and returns the
// non-empty rows of #cf-detail-table. Each row is a slice of trimmed
// cell text strings; rows with no visible td text are skipped (matches
// the Python `if len(row_data) > 0` filter).
func ExtractCFTable(pageHTML string) ([][]string, error) {
	doc, err := html.Parse(strings.NewReader(pageHTML))
	if err != nil {
		return nil, fmt.Errorf("parse cf html: %w", err)
	}
	table := findTableByID(doc, "cf-detail-table")
	if table == nil {
		return nil, fmt.Errorf("cf-detail-table not found")
	}
	return rowsFromTable(table, false), nil
}

// ExtractAssetHistoryTable parses a MoneyForward /bs/history page
// snapshot and returns the asset-history table (header row from thead
// followed by body rows from tbody). Cells containing an <a> tag with
// the text "詳細" are captured as the literal "詳細", and a trailing
// "円" is stripped from numeric cells (matches the Python behaviour).
func ExtractAssetHistoryTable(pageHTML string) ([][]string, error) {
	doc, err := html.Parse(strings.NewReader(pageHTML))
	if err != nil {
		return nil, fmt.Errorf("parse asset html: %w", err)
	}
	table := findTableByClass(doc, "table", "table-bordered")
	if table == nil {
		return nil, fmt.Errorf("asset history table not found")
	}
	return rowsFromTable(table, true), nil
}

func findTableByID(n *html.Node, id string) *html.Node {
	if n == nil {
		return nil
	}
	if n.Type == html.ElementNode && n.Data == "table" {
		for _, a := range n.Attr {
			if a.Key == "id" && a.Val == id {
				return n
			}
		}
	}
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		if t := findTableByID(c, id); t != nil {
			return t
		}
	}
	return nil
}

func findTableByClass(n *html.Node, classes ...string) *html.Node {
	want := map[string]struct{}{}
	for _, c := range classes {
		want[c] = struct{}{}
	}
	var walk func(*html.Node) *html.Node
	walk = func(n *html.Node) *html.Node {
		if n == nil {
			return nil
		}
		if n.Type == html.ElementNode && n.Data == "table" {
			got := map[string]struct{}{}
			for _, a := range n.Attr {
				if a.Key == "class" {
					for _, c := range strings.Fields(a.Val) {
						got[c] = struct{}{}
					}
				}
			}
			all := true
			for c := range want {
				if _, ok := got[c]; !ok {
					all = false
					break
				}
			}
			if all {
				return n
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			if t := walk(c); t != nil {
				return t
			}
		}
		return nil
	}
	return walk(n)
}

func rowsFromTable(table *html.Node, stripYen bool) [][]string {
	headerPart := findChild(table, "thead")
	bodyPart := findChild(table, "tbody")
	var out [][]string
	if headerPart != nil {
		if rows := collectRows(headerPart, stripYen); len(rows) > 0 {
			out = append(out, rows[0])
		}
	}
	if bodyPart != nil {
		out = append(out, collectRows(bodyPart, stripYen)...)
	} else {
		out = append(out, collectRows(table, stripYen)...)
	}
	if !stripYen {
		filtered := out[:0]
		for _, r := range out {
			if rowHasText(r) {
				filtered = append(filtered, r)
			}
		}
		out = filtered
	}
	return out
}

func collectRows(part *html.Node, stripYen bool) [][]string {
	var rows [][]string
	for tr := findChild(part, "tr"); tr != nil; tr = findNextSibling(tr, "tr") {
		var row []string
		hasTd := false
		for cell := findChild(tr, "th"); cell != nil; cell = findNextSibling(cell, "th") {
			row = append(row, cellText(cell, stripYen))
		}
		for cell := findChild(tr, "td"); cell != nil; cell = findNextSibling(cell, "td") {
			row = append(row, cellText(cell, stripYen))
			hasTd = true
		}
		if !stripYen && !hasTd {
			continue
		}
		rows = append(rows, row)
	}
	return rows
}

func cellText(cell *html.Node, stripYen bool) string {
	if a := findChild(cell, "a"); a != nil {
		t := strings.TrimSpace(textOf(a))
		if t == "詳細" {
			return "詳細"
		}
	}
	t := strings.TrimSpace(textOf(cell))
	if stripYen && strings.HasSuffix(t, "円") {
		t = strings.TrimSuffix(t, "円")
	}
	return t
}

func textOf(n *html.Node) string {
	if n == nil {
		return ""
	}
	if n.Type == html.TextNode {
		return n.Data
	}
	var sb strings.Builder
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		sb.WriteString(textOf(c))
	}
	return sb.String()
}

func findChild(n *html.Node, tag string) *html.Node {
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		if c.Type == html.ElementNode && c.Data == tag {
			return c
		}
	}
	return nil
}

func findNextSibling(n *html.Node, tag string) *html.Node {
	for c := n.NextSibling; c != nil; c = c.NextSibling {
		if c.Type == html.ElementNode && c.Data == tag {
			return c
		}
	}
	return nil
}

func rowHasText(row []string) bool {
	for _, c := range row {
		if c != "" {
			return true
		}
	}
	return false
}
