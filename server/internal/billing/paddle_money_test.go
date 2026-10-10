package billing

import "testing"

func TestParsePaddleMinorUnits(t *testing.T) {
	t.Parallel()
	ok := []struct {
		in   string
		want int64
	}{
		{"100", 100},
		{"0", 0},
		{"1", 1},
		{"10000", 10000},
		{"-100", -100},
		{" 100 ", 100},
	}
	for _, tc := range ok {
		tc := tc
		t.Run("ok_"+tc.in, func(t *testing.T) {
			t.Parallel()
			got, err := parsePaddleMinorUnits(tc.in)
			if err != nil {
				t.Fatalf("parsePaddleMinorUnits(%q) err=%v", tc.in, err)
			}
			if got != tc.want {
				t.Fatalf("parsePaddleMinorUnits(%q)=%d want %d", tc.in, got, tc.want)
			}
		})
	}

	bad := []string{"", "1.00", "1.5", "0.01", "-1.00", "10e2", "abc", "12a"}
	for _, in := range bad {
		in := in
		t.Run("bad_"+in, func(t *testing.T) {
			t.Parallel()
			if _, err := parsePaddleMinorUnits(in); err == nil {
				t.Fatalf("parsePaddleMinorUnits(%q) expected error", in)
			}
		})
	}
}
