package cli

import "strings"

// applyHumanChromeFallbacks fills empty human-UX chrome fields with the same
// defaults as server MessageForCode (site_messages). Used when the CLI is newer
// than a deployed API catalog so status/login/start stay usable worldwide.
func applyHumanChromeFallbacks(ch *cliChrome) {
	if ch == nil {
		return
	}
	set := func(dst *string, def string) {
		if strings.TrimSpace(*dst) == "" {
			*dst = def
		}
	}
	set(&ch.StatusLoggedIn, "Logged in")
	set(&ch.StatusLabelPlan, "Plan")
	set(&ch.StatusLabelCredits, "Credits")
	set(&ch.StatusLabelRemaining, "Remaining")
	set(&ch.StatusLabelDashboard, "Dashboard")
	set(&ch.StatusLabelUpgrade, "Upgrade")
	set(&ch.StatusRemainingFmt, "%d / %d")
	set(&ch.StatusChecking, "Checking account…")
	set(&ch.StatusHTTPErrorFmt, "Could not reach Trim Cloud (%s). Check your network and try again.")
	set(&ch.StartBaseURLLabel, "IDE Base URL")
	set(&ch.LoginURLLabel, "Sign-in URL")
	set(&ch.StatusNotLoggedIn, "Not logged in. Run: trim login")
	// Always refresh start banners so attach / cold-start never print a blank line worldwide.
	// Same palette as status/autostart/daemon: Success ● · Table accent/ok · Muted tip.
	ch.StartRunningFmt = "Trim proxy running on %s://127.0.0.1:%s"
	ch.StartAttachedOkFmt = "Trim is already running on %s://127.0.0.1:%s"
	ch.StartAttachedHint = "Attached to the existing local proxy - no second process started."
	ch.StartDashboardLabel = "Dashboard"
	ch.StartDashboardFmt = "Local dashboard: %s://127.0.0.1:%s/dashboard"
	ch.StartPointIDEFmt = "Point your IDE Base URL at %s://127.0.0.1:%s/v1 (Cursor Override / Continue apiBase). VS Code Chat Custom Endpoint needs …/v1/chat/completions - see docs."
	ch.StartBaseURLLabel = "IDE Base URL"

	// Always refresh Windows / OS-hook copy: HKCU Run → stable LocalAppData binary
	// (never node_modules). Ensures worldwide CLIs show correct guidance before API catch-up.
	ch.DaemonWindowsOk = "Windows logon autostart registered (Startup Run → %LOCALAPPDATA%\\Trim\\bin - no admin)"
	ch.DaemonWindowsRemoved = "Windows logon autostart removed"
	ch.DaemonWindowsMissing = "Windows logon autostart is not installed. Run: trim autostart enable"
	ch.DaemonWindowsRunLabel = "Run command"
	ch.AutostartEnabledOk = "Auto-start with IDE enabled"
	ch.AutostartDisabledOk = "Auto-start with IDE disabled"
	ch.AutostartDaemonHint = "OS logon starts Trim for your user account. The Trim IDE extension also attaches to a healthy local proxy."
	ch.AutostartOSHookFailedTitle = "Preference saved, but OS logon autostart could not be registered"
	ch.AutostartOSHookFailedBody = "Your Trim preference is on. IDE auto-start can still work. Fix OS logon registration below, or start manually."
	ch.AutostartOSHookAccessDeniedBody = "Windows blocked the logon registration (Access denied). Trim uses a per-user Startup entry pointing at %LOCALAPPDATA%\\Trim\\bin\\trim.exe (no Administrator on a normal PC). Retry: trim autostart enable. If it still fails, your organization may block Startup programs - use trim start or the Trim IDE extension."
	ch.AutostartOSHookNextStartLabel = "Manual start"
	ch.AutostartOSHookNextIDELabel = "With IDE"
	ch.AutostartOSHookNextIDEValue = "Keep “Start Trim with your IDE” on in the dashboard / extension"
	ch.AutostartOSHookDetailLabel = "Details"
	// Always refresh help so trim help autostart / trim help daemon match stable Windows path + soft-fail UX.
	ch.HelpDaemonShort = "OS logon autostart (launchd / systemd user / Windows Startup Run)"
	ch.HelpDaemonLong = "Advanced OS logon entrypoint (same hook as trim autostart enable).\n\n  trim daemon status\n  trim daemon install\n  trim daemon uninstall\n  trim daemon run\n\nPlatforms:\n  macOS: launchd LaunchAgent (per-user)\n  Linux: systemd --user (per-user)\n  Windows: HKCU Startup Run → %LOCALAPPDATA%\\Trim\\bin\\trim.exe (stable copy outside npm/Scoop; no Administrator on a normal PC)\n\ndaemon run only starts when auto-start is on, and attaches if a healthy proxy is already listening.\nPrefer trim autostart enable for everyday Always-on.\nDocs: https://use-trim.com/docs/cli/autostart"
	ch.HelpDaemonInstallShort = "Install Trim proxy to start on login (launchd / systemd --user / Windows Startup Run)"
	ch.HelpDaemonUninstallShort = "Remove OS logon autostart (Startup Run / launchd / systemd user)"
	ch.HelpDaemonStatusShort = "Show whether OS logon autostart is installed"
	ch.HelpAutostartShort = "Auto-start proxy with your IDE (synced preference)"
	ch.HelpAutostartLong = "Synced Always-on preference (Start Trim with your IDE).\n\n  trim autostart status\n  trim autostart enable\n  trim autostart disable\n\nenable turns the preference on and best-effort registers OS logon:\n  macOS launchd · Linux systemd --user · Windows Startup Run → %LOCALAPPDATA%\\Trim\\bin (no Administrator on a normal PC)\n\nIf OS registration fails, your preference stays on. The CLI prints a yellow warning, muted guidance, and next steps (trim start / IDE). Use -v for raw OS detail.\n\nPrefer this over trim daemon for everyday use.\nDocs: https://use-trim.com/docs/cli/autostart"
	ch.HelpAutostartEnableShort = "Enable auto-start and best-effort OS logon (Windows Startup Run → LocalAppData, no admin)"
	ch.HelpAutostartDisableShort = "Disable auto-start and uninstall OS logon when supported"
}
