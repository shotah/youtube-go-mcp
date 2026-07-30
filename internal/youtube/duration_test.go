package youtube

import (
	"testing"
	"time"
)

func TestParseISO8601Duration(t *testing.T) {
	cases := []struct {
		in   string
		want time.Duration
	}{
		{"PT4M13S", 4*time.Minute + 13*time.Second},
		{"PT1H2M3S", time.Hour + 2*time.Minute + 3*time.Second},
		{"PT45S", 45 * time.Second},
		{"P1DT2H", 26 * time.Hour},
		{"", 0},
		{"P0D", 0},
	}
	for _, tc := range cases {
		if got := parseISO8601Duration(tc.in); got != tc.want {
			t.Errorf("%q: got %v want %v", tc.in, got, tc.want)
		}
	}
}
