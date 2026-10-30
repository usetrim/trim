package cli

import (
	"path/filepath"
	"strings"
	"testing"
)

func TestDetectInstallChannel(t *testing.T) {
	cases := []struct {
		exe  string
		want installChannel
	}{
		{filepath.ToSlash(`C:/Users/x/AppData/Roaming/npm/node_modules/@usetrim/trim/vendor/trim.exe`), channelNPM},
		{filepath.ToSlash(`C:/Users/x/scoop/apps/trim/current/trim.exe`), channelScoop},
		{filepath.ToSlash(`C:/Users/x/scoop/shims/trim.exe`), channelScoop},
		{`/opt/homebrew/Cellar/trim/0.1.21/bin/trim`, channelBrew},
		{filepath.ToSlash(`C:/Users/x/AppData/Local/Microsoft/WinGet/Packages/Trim.CLI_8wekyb3d8bbwe/trim.exe`), channelWinget},
		{filepath.ToSlash(`C:/Users/x/AppData/Local/Trim/bin/trim.exe`), channelStandalone},
		{filepath.ToSlash(`C:/Users/x/AppData/Local/Programs/trim/trim.exe`), channelStandalone},
		{"", channelStandalone},
	}
	for _, tc := range cases {
		got := detectInstallChannel(tc.exe)
		if got != tc.want {
			t.Fatalf("detectInstallChannel(%q)=%q want %q", tc.exe, got, tc.want)
		}
	}
}

func TestNpmPackageRootFromExe(t *testing.T) {
	exe := filepath.FromSlash(`C:/Users/x/AppData/Roaming/npm/node_modules/@usetrim/trim/vendor/trim.exe`)
	root := npmPackageRootFromExe(exe)
	if root == "" {
		t.Fatal("expected npm package root")
	}
	n := strings.ToLower(filepath.ToSlash(root))
	if !strings.HasSuffix(n, "/node_modules/@usetrim/trim") {
		t.Fatalf("root=%q", root)
	}
}
