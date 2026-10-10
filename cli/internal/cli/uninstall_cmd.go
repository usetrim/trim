package cli

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/spf13/cobra"
	"github.com/usetrim/trim/cli/internal/config"
	"github.com/usetrim/trim/cli/internal/storage"
)

// uninstallKeepData skips deleting local data / Deep caches / binary.
// Full purge is the default (pro one-shot uninstall).
var uninstallKeepData bool

// uninstallPurgeLegacy accepts historical --purge; purge is already the default.
var uninstallPurgeLegacy bool

var uninstallCmd = &cobra.Command{
	Use:   "uninstall",
	Short: "",
	Long:  "",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.LoadLocal()
		if err != nil {
			return err
		}
		chrome := fetchCLIChrome(cfg)
		if chrome.UninstallStarting != "" {
			fmt.Println(chrome.UninstallStarting)
		}

		// 1) Stop local proxy (best-effort; may already be down).
		_ = stopCmd.RunE(stopCmd, nil)
		if chrome.UninstallStopped != "" {
			fmt.Println(chrome.UninstallStopped)
		}

		// 2) Remove OS login daemon.
		_ = daemonUninstallCmd.RunE(daemonUninstallCmd, nil)
		if chrome.UninstallDaemonOk != "" {
			fmt.Println(chrome.UninstallDaemonOk)
		}

		// 3) Clear CLI credentials.
		if err := storage.DeleteToken(); err != nil && err != storage.ErrNotLoggedIn {
			return err
		}
		if chrome.UninstallLogoutOk != "" {
			fmt.Println(chrome.UninstallLogoutOk)
		}

		// 4) Revert IDE Base URL overrides written by `trim setup`.
		scheme := "http"
		if strings.TrimSpace(os.Getenv("TRIM_TLS_CERT")) != "" && strings.TrimSpace(os.Getenv("TRIM_TLS_KEY")) != "" {
			scheme = "https"
		}
		trimBase := fmt.Sprintf("%s://localhost:%s/v1", scheme, cfg.Port)
		reverted, _ := revertSetupBaseURLs(trimBase)
		if len(reverted) == 0 {
			if chrome.UninstallSetupNone != "" {
				fmt.Println(chrome.UninstallSetupNone)
			}
		} else if chrome.UninstallSetupRevertedFmt != "" {
			for _, p := range reverted {
				fmt.Printf(chrome.UninstallSetupRevertedFmt+"\n", p)
			}
		}

		// 5) Full purge by default (Windows Apps & features / curl install one-shot).
		if !uninstallKeepData {
			if err := purgeLocalData(chrome); err != nil {
				return err
			}
			if err := purgeDeepMode(chrome); err != nil {
				return err
			}
			purgeSidecars(chrome)
			purgeBinaryAndPath(chrome)
			removeWindowsARP(chrome)
		}

		if chrome.UninstallDone != "" {
			fmt.Println(chrome.UninstallDone)
		}
		printUninstallNextSteps(chrome)
		return nil
	},
}

func printUninstallNextSteps(chrome cliChrome) {
	for _, line := range []string{
		chrome.UninstallNextExt,
		chrome.UninstallNextPkgBrew,
		chrome.UninstallNextPkgScoop,
		chrome.UninstallNextPkgWinget,
	} {
		if strings.TrimSpace(line) != "" {
			fmt.Println(line)
		}
	}
}

func purgeLocalData(chrome cliChrome) error {
	home, err := os.UserHomeDir()
	if err != nil {
		return err
	}
	rels := chromeCSV(chrome.UninstallHomeDirsRel)
	if len(rels) == 0 {
		if chrome.UninstallHomeDirsMissing != "" {
			fmt.Println(chrome.UninstallHomeDirsMissing)
		}
	} else {
		removed := []string{}
		for _, rel := range rels {
			if !safeHomeRel(rel) {
				continue
			}
			dir := filepath.Join(home, filepath.FromSlash(rel))
			if _, err := os.Stat(dir); err != nil {
				continue
			}
			if err := os.RemoveAll(dir); err != nil {
				return err
			}
			removed = append(removed, dir)
		}
		if len(removed) > 0 && chrome.UninstallConfigRemovedFmt != "" {
			fmt.Printf(chrome.UninstallConfigRemovedFmt+"\n", strings.Join(removed, ", "))
		}
	}

	if runtime.GOOS == "darwin" {
		any := false
		for _, rel := range chromeCSV(chrome.UninstallDarwinLogRels) {
			if !safeHomeRel(rel) {
				continue
			}
			p := filepath.Join(home, filepath.FromSlash(rel))
			if err := os.Remove(p); err == nil {
				any = true
			}
		}
		if any && chrome.UninstallLogsRemoved != "" {
			fmt.Println(chrome.UninstallLogsRemoved)
		}
	}
	return nil
}

// safeHomeRel rejects absolute paths and path traversal (fail-closed).
func safeHomeRel(rel string) bool {
	rel = strings.TrimSpace(rel)
	if rel == "" || filepath.IsAbs(rel) || strings.Contains(rel, "..") {
		return false
	}
	clean := filepath.Clean(filepath.FromSlash(rel))
	return clean != "." && clean != ".." && !strings.HasPrefix(clean, "..")
}

func chromeCSV(raw string) []string {
	parts := strings.Split(raw, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}

func purgeDeepMode(chrome cliChrome) error {
	hubDirs := chromeCSV(chrome.UninstallHfHubDirs)
	pipPkgs := chromeCSV(chrome.UninstallPipPackages)
	if len(hubDirs) == 0 && len(pipPkgs) == 0 {
		if chrome.UninstallDeepSkipped != "" {
			fmt.Println(chrome.UninstallDeepSkipped)
		}
		return nil
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return err
	}

	if len(hubDirs) > 0 {
		hfHome := strings.TrimSpace(os.Getenv("HF_HOME"))
		if hfHome == "" {
			rel := strings.TrimSpace(chrome.UninstallHfCacheRel)
			if !safeHomeRel(rel) {
				hubDirs = nil
			} else {
				hfHome = filepath.Join(home, filepath.FromSlash(rel))
			}
		}
		subdir := strings.TrimSpace(chrome.UninstallHfHubSubdir)
		if subdir == "" || subdir != filepath.Base(subdir) || subdir == "." || subdir == ".." {
			hubDirs = nil
		}
		if len(hubDirs) > 0 {
			hubRoot := filepath.Join(hfHome, subdir)
			for _, name := range hubDirs {
				// Fail-closed: only basename segments from DB chrome (no path traversal).
				if name != filepath.Base(name) || name == "." || name == ".." {
					continue
				}
				target := filepath.Join(hubRoot, name)
				_ = os.RemoveAll(target)
			}
		}
	}

	for _, pkg := range pipPkgs {
		pkg = strings.TrimSpace(pkg)
		if pkg == "" || strings.ContainsAny(pkg, " \t;/\\") {
			continue
		}
		for _, bin := range chromeCSV(chrome.UninstallPipBins) {
			if bin != filepath.Base(bin) || bin == "." || bin == ".." {
				continue
			}
			_ = exec.Command(bin, "uninstall", "-y", pkg).Run()
		}
		mod := strings.TrimSpace(chrome.UninstallPipModule)
		if mod != "" && mod == filepath.Base(mod) && mod != "." && mod != ".." {
			for _, bin := range chromeCSV(chrome.UninstallPythonBins) {
				if bin != filepath.Base(bin) || bin == "." || bin == ".." {
					continue
				}
				_ = exec.Command(bin, "-m", mod, "uninstall", "-y", pkg).Run()
			}
		}
	}

	if chrome.UninstallDeepOk != "" {
		fmt.Println(chrome.UninstallDeepOk)
	}
	return nil
}

func purgeSidecars(chrome cliChrome) {
	exe, err := os.Executable()
	if err != nil {
		return
	}
	exe, err = filepath.Abs(exe)
	if err != nil {
		return
	}
	dir := filepath.Dir(exe)
	for _, name := range chromeCSV(chrome.UninstallSidecarNames) {
		base := filepath.Base(name)
		if base == "." || base == ".." {
			continue
		}
		path := filepath.Join(dir, base)
		if err := os.RemoveAll(path); err == nil && chrome.UninstallSidecarRemovedFmt != "" {
			fmt.Printf(chrome.UninstallSidecarRemovedFmt+"\n", path)
		}
	}
}

func purgeBinaryAndPath(chrome cliChrome) {
	exe, err := os.Executable()
	if err != nil {
		return
	}
	exe, err = filepath.Abs(exe)
	if err != nil {
		return
	}
	installDir := filepath.Dir(exe)

	if err := os.Remove(exe); err != nil {
		if chrome.UninstallBinaryManualFmt != "" {
			fmt.Printf(chrome.UninstallBinaryManualFmt+"\n", exe)
		}
	} else if chrome.UninstallBinaryRemovedFmt != "" {
		fmt.Printf(chrome.UninstallBinaryRemovedFmt+"\n", exe)
	}

	if runtime.GOOS == "windows" {
		if removeWindowsUserPathEntry(installDir) && chrome.UninstallPathRemoved != "" {
			fmt.Println(chrome.UninstallPathRemoved)
		}
		// Best-effort: remove empty install dir after binary gone.
		_ = os.Remove(installDir)
	}
}

func removeWindowsUserPathEntry(dir string) bool {
	if runtime.GOOS != "windows" {
		return false
	}
	dir = filepath.Clean(dir)
	cmd := exec.Command("powershell", "-NoProfile", "-Command",
		`$dir = $env:TRIM_UNINSTALL_DIR; if (-not $dir) { exit 1 };
$userPath = [Environment]::GetEnvironmentVariable('Path','User');
if (-not $userPath) { exit 0 };
$parts = $userPath -split ';' | Where-Object { $_ -and ($_.TrimEnd('\') -ne $dir.TrimEnd('\')) };
[Environment]::SetEnvironmentVariable('Path', ($parts -join ';'), 'User');
`)
	cmd.Env = append(os.Environ(), "TRIM_UNINSTALL_DIR="+dir)
	return cmd.Run() == nil
}

func removeWindowsARP(chrome cliChrome) {
	if runtime.GOOS != "windows" {
		return
	}
	key := strings.TrimSpace(chrome.UninstallArpRegKey)
	if key == "" || strings.ContainsAny(key, `/\`) || key == "." || key == ".." {
		return
	}
	cmd := exec.Command("powershell", "-NoProfile", "-Command",
		`$k = $env:TRIM_ARP_KEY; if (-not $k) { exit 1 };
$path = Join-Path 'HKCU:\Software\Microsoft\Windows\CurrentVersion\Uninstall' $k;
if (Test-Path $path) { Remove-Item -Path $path -Recurse -Force }
`)
	cmd.Env = append(os.Environ(), "TRIM_ARP_KEY="+key)
	if cmd.Run() == nil && chrome.UninstallArpRemoved != "" {
		fmt.Println(chrome.UninstallArpRemoved)
	}
}

// ensureWindowsARPRegistration writes HKCU Uninstall so Settings → Apps runs `trim uninstall`.
// DisplayName / Publisher / RegKey come from site_messages chrome only (fail-closed).
func ensureWindowsARPRegistration(chrome cliChrome) {
	if runtime.GOOS != "windows" {
		return
	}
	display := strings.TrimSpace(chrome.UninstallArpDisplayName)
	publisher := strings.TrimSpace(chrome.UninstallArpPublisher)
	key := strings.TrimSpace(chrome.UninstallArpRegKey)
	if display == "" || publisher == "" || key == "" {
		return
	}
	if strings.ContainsAny(key, `/\`) || key == "." || key == ".." {
		return
	}
	exe, err := os.Executable()
	if err != nil {
		return
	}
	exe, err = filepath.Abs(exe)
	if err != nil {
		return
	}
	installDir := filepath.Dir(exe)
	cmd := exec.Command("powershell", "-NoProfile", "-Command",
		`$key = $env:TRIM_ARP_KEY; $display = $env:TRIM_ARP_DISPLAY; $publisher = $env:TRIM_ARP_PUBLISHER;
$exe = $env:TRIM_ARP_EXE; $dir = $env:TRIM_ARP_DIR;
if (-not $key -or -not $display -or -not $publisher -or -not $exe) { exit 1 };
$path = Join-Path 'HKCU:\Software\Microsoft\Windows\CurrentVersion\Uninstall' $key;
New-Item -Path $path -Force | Out-Null;
Set-ItemProperty -Path $path -Name 'DisplayName' -Value $display;
Set-ItemProperty -Path $path -Name 'Publisher' -Value $publisher;
Set-ItemProperty -Path $path -Name 'InstallLocation' -Value $dir;
Set-ItemProperty -Path $path -Name 'UninstallString' -Value ('"{0}" uninstall' -f $exe);
Set-ItemProperty -Path $path -Name 'QuietUninstallString' -Value ('"{0}" uninstall' -f $exe);
Set-ItemProperty -Path $path -Name 'NoModify' -Value 1 -Type DWord;
Set-ItemProperty -Path $path -Name 'NoRepair' -Value 1 -Type DWord;
`)
	cmd.Env = append(os.Environ(),
		"TRIM_ARP_KEY="+key,
		"TRIM_ARP_DISPLAY="+display,
		"TRIM_ARP_PUBLISHER="+publisher,
		"TRIM_ARP_EXE="+exe,
		"TRIM_ARP_DIR="+installDir,
	)
	_ = cmd.Run()
}

func init() {
	uninstallCmd.Flags().BoolVar(&uninstallKeepData, "keep-data", false, "")
	// Historical alias: full purge is already the default.
	uninstallCmd.Flags().BoolVar(&uninstallPurgeLegacy, "purge", false, "")
	_ = uninstallCmd.Flags().MarkHidden("purge")
	_ = uninstallPurgeLegacy
}
