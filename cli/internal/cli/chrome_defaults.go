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
	ch.HelpStartLong = "Starts the local proxy for IDE traffic, then returns your shell (proxy keeps running in the background). Stop with: trim stop. Use --foreground to keep this terminal attached (Ctrl+C stops).\n\nFast Mode always runs. When account preferences set Deep Mode on, each request also runs Deep (LLMLingua) after Fast before forwarding upstream.\n\nAfter traffic flows, check savings with:\n  trim stats\n  http://127.0.0.1:<port>/dashboard → Show savings detail"

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

	// Always refresh root / config / uninstall / Cursor Link help so a CLI newer than
	// the deployed site_messages catalog still lists cursor-link, config sync, and
	// package-manager uninstall (npm -g) correctly worldwide.
	ch.HelpRootLong = "Trim keeps a local Fast Mode proxy on your machine so IDE traffic uses less noisy context.\n\nEveryday: start, stop, status, stats, upgrade, autostart, cursor-link, login, logout, uninstall, setup, version\nAdvanced: tui, compress, daemon, telemetry, config (get | set | sync)\n\nCursor Agent: trim cursor-link enable → trim start → trim cursor-link url (paste HTTPS …/v1; not 127.0.0.1).\n\nSimple savings check after trim start:\n  trim stats\n  open http://127.0.0.1:<port>/dashboard → Show savings detail\n\ntrim status = cloud quota. trim stats = local Deep status / savings meter.\n\nFull remove: trim uninstall (then brew/scoop/winget/npm uninstall if you installed that way).\n\nRun \"trim help <command>\" for details. Prefer trim start / autostart for daily use; compress and Deep Mode are file/batch tools, not the everyday IDE path."
	ch.HelpConfigShort = "Local Fast/Deep preferences (get | set | sync)"
	ch.HelpConfigLong = "Manage local Fast/Deep preferences under ~/.config/trim/.\n\n  trim config get\n  trim config get default-tier\n  trim config set default-tier deep\n  trim config sync\n\nget - show local preferences\nset - write a local preference key\nsync - pull Fast/Deep preferences and provider adapters from your Trim cloud profile (requires trim login)"
	ch.HelpConfigGetShort = "Show preferences (default-tier, deep-engine, target-token)"
	ch.HelpConfigSetShort = "Set a preference (example: trim config set default-tier deep)"
	ch.HelpConfigSyncShort = "Pull Fast/Deep preferences and provider adapters from your Trim cloud profile into ~/.config/trim/"
	ch.HelpUninstallShort = "Fully remove Trim from this machine (proxy, daemon, data, Deep caches, binary)"
	ch.HelpUninstallLong = "Fully removes Trim from this machine:\n  - stops the local proxy and Cursor Link tunnel\n  - removes the OS login daemon\n  - clears CLI credentials\n  - reverts IDE Base URL overrides from trim setup\n  - deletes ~/.trim and ~/.config/trim, Deep caches, sidecars, and the CLI binary/PATH when possible\n\nUse --keep-data to skip deleting local data directories, Deep caches, and the binary.\n\nIf you installed via a package manager, also uninstall that package after trim uninstall:\n  brew uninstall trim\n  scoop uninstall trim\n  winget uninstall Trim.CLI\n  npm uninstall -g @usetrim/trim"
	ch.UninstallNextPkgNpm = "If installed via npm: npm uninstall -g @usetrim/trim"
	ch.HelpCursorLinkShort = "Public HTTPS URL to local Trim for Cursor Agent (managed tunnel)"
	ch.HelpCursorLinkLong = "Trim Cursor Link exposes your local Trim proxy through an automatic Cloudflare quick tunnel so Cursor Agent (cloud) can reach it without 127.0.0.1.\n\nCompression, meters, and adapters still run on this PC. The public hostname is only a door. Lifecycle matches trim start / trim stop when enabled.\n\nTypical flow:\n  trim cursor-link enable\n  trim start\n  trim cursor-link url\n  # paste …/cl/<token>/v1 into Cursor Override OpenAI Base URL\n  trim stop\n\nSubcommands: enable, disable, status, url, rotate"
	ch.HelpCursorLinkEnableShort = "Enable Cursor Link (auto tunnel while trim start is running)"
	ch.HelpCursorLinkDisableShort = "Disable Cursor Link and stop any running tunnel"
	ch.HelpCursorLinkStatusShort = "Show Cursor Link preference and live public URL"
	ch.HelpCursorLinkURLShort = "Print the live Cursor Agent Base URL (…/v1) for copy/paste"
	ch.HelpCursorLinkRotateShort = "Rotate the Cursor Link access token (revokes the previous public URL path)"
}
