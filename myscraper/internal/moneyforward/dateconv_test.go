package moneyforward

import (
	"testing"
	"time"
)

func TestConvertDateField(t *testing.T) {
	cases := []struct {
		name      string
		dateText  string
		now       time.Time
		lastmonth bool
		want      string
	}{
		{
			name:      "march current month keeps year",
			dateText:  "03/12（＊）",
			now:       time.Date(2024, 3, 14, 0, 0, 0, 0, time.UTC),
			lastmonth: false,
			want:      "2024/03/12",
		},
		{
			name:      "march lastmonth keeps year",
			dateText:  "02/12（＊）",
			now:       time.Date(2024, 3, 14, 0, 0, 0, 0, time.UTC),
			lastmonth: true,
			want:      "2024/02/12",
		},
		{
			name:      "december current month keeps year",
			dateText:  "12/12（＊）",
			now:       time.Date(2024, 12, 14, 0, 0, 0, 0, time.UTC),
			lastmonth: false,
			want:      "2024/12/12",
		},
		{
			name:      "january lastmonth rolls back to previous year",
			dateText:  "12/17（＊）",
			now:       time.Date(2024, 1, 3, 0, 0, 0, 0, time.UTC),
			lastmonth: true,
			want:      "2023/12/17",
		},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			got := ConvertDateField(tc.dateText, tc.lastmonth, tc.now)
			if got != tc.want {
				t.Fatalf("ConvertDateField(%q, %v, %v) = %q, want %q", tc.dateText, tc.lastmonth, tc.now, got, tc.want)
			}
		})
	}
}
