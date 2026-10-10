package catalogsync

import (
	"fmt"
	"testing"
)

func TestPaddleNotFound(t *testing.T) {
	t.Parallel()
	cases := []struct {
		msg  string
		want bool
	}{
		{"", false},
		{"paddle api PATCH /products/x: status=404 code=not_found detail=pro_x Product not found.", true},
		{"paddle api GET /prices/x: Price pri_x not found.", true},
		{"paddle api POST /products: status=403 code=forbidden detail=You aren't permitted", false},
		{"connection refused", false},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.msg, func(t *testing.T) {
			t.Parallel()
			var err error
			if tc.msg != "" {
				err = fmt.Errorf("%s", tc.msg)
			}
			if got := paddleNotFound(err); got != tc.want {
				t.Fatalf("paddleNotFound(%q)=%v want %v", tc.msg, got, tc.want)
			}
		})
	}
}
