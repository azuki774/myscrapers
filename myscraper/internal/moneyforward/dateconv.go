// Package moneyforward implements the MoneyForward (家計簿) scrape flow that
// used to live in src/moneyforward/main.py. The package is split into pure
// helpers (dateconv, csvrow, cookies, htmltable), a Session interface that
// hides the browser, and orchestration entry points (Fetch, Update) plus an
// S3 uploader.
package moneyforward

import (
	"strconv"
	"time"
)

// ConvertDateField maps a MoneyForward "MM/DD(曜日)" cell into a "YYYY/MM/DD"
// string. When lastmonth is true and the cell is December, the year rolls
// back by one (the Python script's edge case for January viewing the
// December prior year).
func ConvertDateField(dateText string, lastmonth bool, now time.Time) string {
	year := strconv.Itoa(now.Year())
	month := dateText[0:2]
	if lastmonth && month == "12" {
		year = strconv.Itoa(now.Year() - 1)
	}
	return year + "/" + dateText[0:5]
}
