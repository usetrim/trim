package deepopt

import "testing"

func TestFormatSavingPercent(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name       string
		origin     int
		compressed int
		want       string
	}{
		{name: "large_cut", origin: 2391, compressed: 61, want: "97.4%"},
		{name: "exact_half", origin: 100, compressed: 50, want: "50.0%"},
		{name: "no_change", origin: 100, compressed: 100, want: "0.0%"},
		{name: "grew", origin: 51, compressed: 54, want: "-5.9%"},
		{name: "zero_origin", origin: 0, compressed: 0, want: "0.0%"},
		{name: "full_cut", origin: 80, compressed: 0, want: "100.0%"},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := FormatSavingPercent(tc.origin, tc.compressed)
			if got != tc.want {
				t.Fatalf("FormatSavingPercent(%d,%d)=%q want %q", tc.origin, tc.compressed, got, tc.want)
			}
		})
	}
}
