package cli

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// installChannel is how the running CLI binary was distributed.
type installChannel string

const (
	channelStandalone installChannel = "standalone"
	channelNPM        installChannel = "npm"
	channelScoop      installChannel = "scoop"
	channelBrew       installChannel = "brew"
	channelWinget     installChannel = "winget"
)

const uninstallMarkerName = ".trim-uninstalling"

// detectInstallChannel classifies the running executable path (fail-closed to standalone).
func detectInstallChannel(exe string) installChannel {
	n := strings.ToLower(filepath.ToSlash(strings.TrimSpace(exe)))
	if n == "" {
		return channelStandalone
	}
	switch {
	case strings.Contains(n, "/node_modules/@usetrim/trim/"),
		strings.Contains(n, "/node_modules/@usetrim/trim"):
		return channelNPM
	case strings.Contains(n, "/scoop/apps/trim/"),
		strings.Contains(n, "/scoop/shims/trim"):
		return channelScoop
	case strings.Contains(n, "/cellar/trim/"):
		return channelBrew
	case strings.Contains(n, "/microsoft/winget/packages/") && strings.Contains(n, "trim"),
		strings.Contains(n, "/microsoft/winget/links/trim"):
		return channelWinget
	default:
		return channelStandalone
	}
}

// detectPkgChannels finds every package manager that still owns a Trim install,
// not only the channel of the running exe (dual curl+npm / Links shim cases).
func detectPkgChannels(exe string) []installChannel {
	seen := map[installChannel]bool{}
	out := []installChannel{}
	add := func(ch installChannel) {
		if !channelNeedsPkgUninstall(ch) || seen[ch] {
			return
		}
		seen[ch] = true
		out = append(out, ch)
	}
	add(detectInstallChannel(exe))

	// npm global package still present (self-heal risk even if running LocalAppData copy).
	if npmGlobalHasUsetrimTrim() {
		add(channelNPM)
	}
	if commandExists("scoop") && scoopHasTrim() {
		add(channelScoop)
	}
	if commandExists("brew") && brewHasTrim() {
		add(channelBrew)
	}
	if commandExists("winget") && wingetHasTrim() {
		add(channelWinget)
	}
	return out
}

func channelNeedsPkgUninstall(ch installChannel) bool {
	switch ch {
	case channelNPM, channelScoop, channelBrew, channelWinget:
		return true
	default:
		return false
	}
}

func channelPkgLabel(ch installChannel) string {
	switch ch {
	case channelNPM:
		return "npm uninstall -g @usetrim/trim"
	case channelScoop:
		return "scoop uninstall trim"
	case channelBrew:
		return "brew uninstall trim"
	case channelWinget:
		return "winget uninstall Trim.CLI"
	default:
		return string(ch)
	}
}

func commandExists(name string) bool {
	_, err := exec.LookPath(name)
	return err == nil
}

func npmGlobalHasUsetrimTrim() bool {
	if !commandExists("npm") {
		return false
	}
	// Preferred: resolve global root and Stat the package directory (works even
	// when `npm ls` exits non-zero or prints warnings only).
	if out, err := exec.Command("npm", "root", "-g").CombinedOutput(); err == nil {
		root := strings.TrimSpace(string(out))
		if root != "" {
			pkg := filepath.Join(root, "@usetrim", "trim")
			if st, err := os.Stat(pkg); err == nil && st.IsDir() {
				return true
			}
		}
	}
	out, err := exec.Command("npm", "ls", "-g", "--depth=0", "--parseable", "@usetrim/trim").CombinedOutput()
	if err != nil {
		// npm ls exits non-zero when missing; also check stdout path hint.
		s := string(out)
		return strings.Contains(strings.ToLower(s), "@usetrim"+string(os.PathSeparator)+"trim") ||
			strings.Contains(strings.ToLower(filepath.ToSlash(s)), "@usetrim/trim")
	}
	s := strings.TrimSpace(string(out))
	return s != "" && (strings.Contains(filepath.ToSlash(strings.ToLower(s)), "@usetrim/trim") ||
		strings.Contains(strings.ToLower(s), "@usetrim"+string(os.PathSeparator)+"trim"))
}

func scoopHasTrim() bool {
	out, err := exec.Command("scoop", "list", "trim").CombinedOutput()
	if err != nil {
		return false
	}
	// Require a line that looks like the trim app, not any substring "trim".
	for _, line := range strings.Split(string(out), "\n") {
		fields := strings.Fields(strings.ToLower(line))
		if len(fields) > 0 && fields[0] == "trim" {
			return true
		}
	}
	return false
}

func brewHasTrim() bool {
	err := exec.Command("brew", "list", "--formula", "trim").Run()
	return err == nil
}

func wingetHasTrim() bool {
	out, err := exec.Command("winget", "list", "--id", "Trim.CLI", "-e").CombinedOutput()
	if err != nil {
		return strings.Contains(strings.ToLower(string(out)), "trim.cli")
	}
	return strings.Contains(strings.ToLower(string(out)), "trim.cli")
}

// writeUninstallMarkers blocks npm self-heal mid-uninstall via a package-local
// marker only. TEMP markers are intentionally not used: if a post-exit helper
// never runs, a TEMP file would stick forever and break future installs.
func writeUninstallMarkers(exe string) {
	if npmRoot := npmPackageRootFromExe(exe); npmRoot != "" {
		_ = os.WriteFile(filepath.Join(npmRoot, uninstallMarkerName), []byte("1"), 0o600)
	}
	// Also mark any discovered global npm package root.
	if out, err := exec.Command("npm", "root", "-g").CombinedOutput(); err == nil {
		root := strings.TrimSpace(string(out))
		if root != "" {
			pkg := filepath.Join(root, "@usetrim", "trim")
			if st, err := os.Stat(pkg); err == nil && st.IsDir() {
				_ = os.WriteFile(filepath.Join(pkg, uninstallMarkerName), []byte("1"), 0o600)
			}
		}
	}
}

func npmPackageRootFromExe(exe string) string {
	n := filepath.ToSlash(exe)
	const needle = "/node_modules/@usetrim/trim/"
	idx := strings.Index(strings.ToLower(n), needle)
	if idx < 0 {
		return ""
	}
	// Reconstruct with original casing length.
	end := idx + len("/node_modules/@usetrim/trim")
	if end > len(n) {
		return ""
	}
	return filepath.FromSlash(n[:end])
}

func clearUninstallMarkers(exe string) {
	// Legacy TEMP marker cleanup (older builds); no longer written.
	_ = os.Remove(filepath.Join(os.TempDir(), "trim-uninstalling"))
	if npmRoot := npmPackageRootFromExe(exe); npmRoot != "" {
		_ = os.Remove(filepath.Join(npmRoot, uninstallMarkerName))
	}
	if out, err := exec.Command("npm", "root", "-g").CombinedOutput(); err == nil {
		root := strings.TrimSpace(string(out))
		if root != "" {
			_ = os.Remove(filepath.Join(root, "@usetrim", "trim", uninstallMarkerName))
		}
	}
}
