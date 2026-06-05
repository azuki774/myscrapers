package browser

import (
	"os"
	"reflect"
	"testing"
)

func TestChromiumLaunchArgs(t *testing.T) {
	cases := []struct {
		name string
		set  bool
		env  string
		want []string
	}{
		{
			name: "unset env var returns nil",
			set:  false,
			want: nil,
		},
		{
			name: "single arg",
			set:  true,
			env:  "--no-sandbox",
			want: []string{"--no-sandbox"},
		},
		{
			name: "multiple args separated by whitespace",
			set:  true,
			env:  "--no-sandbox  --disable-dev-shm-usage",
			want: []string{"--no-sandbox", "--disable-dev-shm-usage"},
		},
		{
			name: "tabs and newlines are also separators",
			set:  true,
			env:  "--no-sandbox\t--disable-dev-shm-usage\n--disable-gpu",
			want: []string{"--no-sandbox", "--disable-dev-shm-usage", "--disable-gpu"},
		},
		{
			name: "whitespace-only value yields nil so the launch options stay minimal",
			set:  true,
			env:  "   ",
			want: nil,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			const key = "MYSCRAPER_CHROMIUM_ARGS"
			os.Unsetenv(key)
			t.Cleanup(func() { os.Unsetenv(key) })
			if tc.set {
				t.Setenv(key, tc.env)
			}

			got := chromiumLaunchArgs()
			if !reflect.DeepEqual(got, tc.want) {
				t.Fatalf("chromiumLaunchArgs() = %#v, want %#v", got, tc.want)
			}
		})
	}
}
