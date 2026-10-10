package paddleapi

import "testing"

func TestCatalogWriteReady(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name    string
		key     string
		wantErr bool
	}{
		{name: "empty", key: "", wantErr: true},
		{name: "placeholder xxx whole", key: "pdl_xxx", wantErr: true},
		{name: "example dots", key: "pdl_sdbx_apikey_...", wantErr: true},
		{name: "client token shape", key: "live_abc123", wantErr: true},
		{name: "missing apikey segment", key: "pdl_live_notaserverkey_01abc", wantErr: true},
		// Real keys are random; must not reject just because "xxx" appears inside.
		{name: "live key with xxx substring", key: "pdl_live_apikey_01hxxxyyy_secretpart", wantErr: false},
		{name: "live key normal", key: "pdl_live_apikey_01m4gfd9ah7xwv4x659jg0b5n9_secret", wantErr: false},
		{name: "sandbox key", key: "pdl_sdbx_apikey_01abc_secret", wantErr: false},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			c := &Client{APIKey: tc.key}
			err := c.CatalogWriteReady()
			if tc.wantErr && err == nil {
				t.Fatalf("expected error for key shape %q", tc.name)
			}
			if !tc.wantErr && err != nil {
				t.Fatalf("unexpected error for %q: %v", tc.name, err)
			}
		})
	}
}
