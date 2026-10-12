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

	// Always refresh Windows / OS-hook copy: behavior moved from schtasks → HKCU Run.
	// Ensures worldwide CLIs show correct guidance even before API site_messages catch up.
	ch.DaemonWindowsOk = "Windows logon autostart registered (per-user Startup Run key - no admin required)"
	ch.DaemonWindowsRemoved = "Windows logon autostart removed"
	ch.DaemonWindowsMissing = "Windows logon autostart is not installed. Run: trim autostart enable"
	ch.DaemonWindowsRunLabel = "Run command"
	ch.AutostartEnabledOk = "Auto-start with IDE enabled"
	ch.AutostartDisabledOk = "Auto-start with IDE disabled"
	ch.AutostartDaemonHint = "OS logon starts Trim for your user account. The Trim IDE extension also attaches to a healthy local proxy."
	ch.AutostartOSHookFailedTitle = "Preference saved, but OS logon autostart could not be registered"
	ch.AutostartOSHookFailedBody = "Your Trim preference is on. IDE auto-start can still work. Fix OS logon registration below, or start manually."
	ch.AutostartOSHookAccessDeniedBody = "Windows blocked the logon registration (Access denied). Trim now uses a per-user Startup entry that normally needs no Administrator. Retry: trim autostart enable. If it still fails, your organization may block Startup programs - use trim start or the Trim IDE extension."
	ch.AutostartOSHookNextStartLabel = "Manual start"
	ch.AutostartOSHookNextIDELabel = "With IDE"
	ch.AutostartOSHookNextIDEValue = "Keep “Start Trim with your IDE” on in the dashboard / extension"
	ch.AutostartOSHookDetailLabel = "Details"
	ch.HelpDaemonInstallShort = "Install Trim proxy to start on login (launchd / systemd --user / Windows Startup Run)"
}
