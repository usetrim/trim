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
	set(&ch.LoginURLLabel, "Sign-in URL")
	set(&ch.StatusNotLoggedIn, "Not logged in. Run: trim login")
	// Always refresh start banners so attach / cold-start never print a blank line worldwide.
	// Same palette as status/autostart/daemon: Success ● · Table accent/ok · Muted tip.
	ch.StartRunningFmt = "Trim proxy running on %s://127.0.0.1:%s"
	ch.StartAttachedOkFmt = "Trim is already running on %s://127.0.0.1:%s"
	ch.StartAttachedHint = "Attached to the existing local proxy - no second process started."
	ch.StartDashboardLabel = "Dashboard"
	ch.StartDashboardFmt = "Local dashboard: %s://127.0.0.1:%s/dashboard"
	ch.StartBaseURLLabel = "Local Base URL"
	ch.StartPointIDEFmt = "On-device clients: Continue apiBase / VS Code Chat Custom Endpoint / Claude Code origin use %s://127.0.0.1:%s (Continue and most SDKs need …/v1; VS Code Chat needs …/v1/chat/completions; Claude Code needs origin only, no /v1)."
	ch.StartCursorAgentLabel = "Cursor Agent Base URL"
	ch.StartCursorAgentHintOff = "Cursor Agent: do not paste 127.0.0.1 (private networks forbidden). Run: trim cursor-link enable → trim cursor-link url → paste HTTPS …/v1 into Cursor Override OpenAI Base URL. Docs: https://use-trim.com/docs/cli/cursor-link"
	ch.StartCursorAgentTipOn = "Cursor Agent: paste Cursor Agent Base URL into Override OpenAI Base URL (HTTPS …/v1, not 127.0.0.1)."
	ch.StartCursorAgentLocalTip = "Local tools / Claude Code: keep using http://127.0.0.1:%s as documented."
	ch.StartCursorAgentCloudflaredTip = "Trim auto-downloads cloudflared to ~/.trim/bin when missing (all OS). Optional: brew/winget only if installed. Or set TRIM_CLOUDFLARED, then: trim cursor-link status"
	ch.SetupCursorAgentHeader = "Cursor Agent (cloud) - use Trim Cursor Link, not localhost:"
	ch.SetupCursorAgentBlock = "  trim cursor-link enable\n  trim start\n  trim cursor-link url\n  # paste HTTPS …/cl/<token>/v1 into Cursor → Settings → Models → Override OpenAI Base URL\n  # do not paste http://127.0.0.1:…/v1 for Cursor Agent\n  Docs: https://use-trim.com/docs/cli/cursor-link"
	ch.SetupStepCursorLink = "Cursor Agent: trim cursor-link enable && trim start && trim cursor-link url (paste HTTPS …/v1; not 127.0.0.1)"
	ch.SetupCursorSkipped = "Cursor settings.json: skipped writing localhost Base URL (Cursor Agent needs Trim Cursor Link). Continue / VS Code / Windsurf / Zed still use local …/v1 when present."
	ch.HelpStartLong = "Starts the local proxy for IDE traffic, then returns your shell (proxy keeps running in the background). Stop with: trim stop. Use --foreground to keep this terminal attached (Ctrl+C stops).\n\nFast Mode always runs. When account preferences set Deep Mode on, each request also runs Deep (LLMLingua) after Fast before forwarding upstream.\n\nLocal Base URL (127.0.0.1) is for on-device clients (Continue, VS Code Chat, Claude Code). Cursor Agent needs Trim Cursor Link: trim cursor-link enable → trim cursor-link url (HTTPS …/v1; not 127.0.0.1).\n\nAfter traffic flows, check savings with:\n  trim stats\n  http://127.0.0.1:<port>/dashboard → Show savings detail"

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
	ch.HelpRootLong = "Trim keeps a local Fast Mode proxy on your machine so IDE traffic uses less noisy context.\n\nEveryday: start, stop, status, stats, upgrade, autostart, cursor-link, login, logout, uninstall, setup, version\nAdvanced: tui, compress, daemon, telemetry, config (get | set | sync | rates | models)\n\nFast vs Deep (new accounts default to Deep = Fast then Deep):\n  trim config get default-tier\n  trim config set default-tier deep\n  trim config set default-tier fast\n  trim config rates\n  trim config set compression-rate 0.7\n  trim config models\n  trim config set deep-v2-backbone mbert-base\n  trim config sync\n\nDeep keep-rate (after/before, Microsoft LLMLingua): Dashboard slider or trim config rates / set compression-rate (billing min/max/step).\nDeep Base Model (Microsoft LLMLingua-2): Dashboard picker or trim config models / set deep-v2-backbone (billing catalog; default mbert-base, optional xlm-roberta-large).\n\nCursor Agent: trim cursor-link enable → trim start → trim cursor-link url (paste HTTPS …/v1; not 127.0.0.1).\n\nSimple savings check after trim start:\n  trim stats\n  open http://127.0.0.1:<port>/dashboard → Show savings detail\n\ntrim status = cloud quota. trim stats = local Deep status / savings meter.\n\nFull remove: trim uninstall (stops Trim processes, deletes local data/binary, and auto-removes npm/Scoop/Homebrew/winget installs when detected).\n\nRun \"trim help <command>\" for details. Prefer trim start / autostart for daily use (new accounts default Deep = Fast then Deep on the live proxy). Use trim compress --deep for file/batch."
	ch.HelpConfigShort = "Local Fast/Deep preferences (get | set | sync | rates | models)"
	ch.HelpConfigLong = "Manage local Fast/Deep preferences under ~/.config/trim/.\n\nNew accounts default to Deep Mode (live proxy runs Fast then Deep). Switch to Fast anytime.\n\n  trim config get\n  trim config get default-tier\n  trim config get compression-rate\n  trim config get deep-v2-backbone\n  trim config rates\n  trim config models\n  trim config set default-tier deep\n  trim config set default-tier fast\n  trim config set compression-rate 0.7\n  trim config set deep-v2-backbone mbert-base\n  trim config set deep-v2-backbone xlm-roberta-large\n  trim config sync\n\nget - show local preferences\nrates - list allowed keep-rates; mark billing default and selected\nmodels - list allowed LLMLingua-2 Base Models; mark billing default and selected\nset - write a local preference key (when logged in, default-tier / deep-engine / compression-rate / deep-v2-backbone also update your Trim cloud profile)\nsync - pull Fast/Deep preferences and provider adapters from your Trim cloud profile (requires trim login)\n\nFast = live proxy Fast only. Deep = live proxy runs Fast then Deep (LLMLingua-2) at your compression rate and Base Model.\nAfter Dashboard → Settings changes: trim config sync, then restart with trim start."
	ch.HelpConfigGetShort = "Show preferences (default-tier, deep-engine, deep-v2-backbone, compression-rate)"
	ch.HelpConfigSetShort = "Set a preference (example: trim config set deep-v2-backbone mbert-base)"
	ch.HelpConfigSyncShort = "Pull Fast/Deep preferences and provider adapters from your Trim cloud profile into ~/.config/trim/"
	ch.HelpConfigRatesShort = "List allowed Deep compression rates; mark billing default and selected"
	ch.HelpConfigModelsShort = "List allowed Deep v2 Base Models; mark billing default and selected"
	ch.HelpCompressFlagRate = "Deep keep-rate after/before (Microsoft LLMLingua); omit to use synced preferences"
	ch.HelpCompressFlagTarget = ch.HelpCompressFlagRate
	ch.ConfigGetRateFmt = "compression-rate: %g"
	ch.ConfigGetBackboneFmt = "deep-v2-backbone: %s"
	ch.ConfigUnknownKeyFmt = "unknown key %q (default-tier|deep-engine|compression-rate|deep-v2-backbone)"
	ch.ConfigRateRange = "compression-rate must be within billing deep_compression_rate_min..max (typically 0.3 to 1.0)"
	ch.ConfigBackboneInvalid = "deep-v2-backbone must be an id from the synced Base Model catalog (trim config models)"
	ch.ConfigSyncedFmt = "Synced preferences: tier=%s engine=%s backbone=%s compression-rate=%g"
	ch.ConfigRatesTitle = "Compression rates (after / before)"
	ch.ConfigRatesDefaultLabel = "billing default"
	ch.ConfigRatesSelectedLabel = "selected"
	ch.ConfigRatesSelectedMark = "selected"
	ch.ConfigRatesSyncHint = "Bounds and step come from Admin billing via trim config sync. Set with: trim config set compression-rate <rate>"
	ch.ConfigRatesUnsynced = "Rate bounds not synced. Run: trim config sync"
	ch.ConfigModelsTitle = "Deep v2 Base Models (LLMLingua-2)"
	ch.ConfigModelsDefaultLabel = "billing default"
	ch.ConfigModelsSelectedLabel = "selected"
	ch.ConfigModelsSelectedMark = "selected"
	ch.ConfigModelsSyncHint = "Catalog comes from Admin billing via trim config sync. Set with: trim config set deep-v2-backbone <id>"
	ch.ConfigModelsUnsynced = "Base Model catalog not synced. Run: trim config sync"
	ch.ConfigModelsHfLabel = "hf-model"
	// Prefer site_messages from API chrome. Fill only when empty so Admin DB edits win.
	set(&ch.HelpCompressShort, "Compress a file (Fast or Deep). Live IDE proxy also runs Deep when Preferences Deep is on (v2 uses synced Base Model).")
	set(&ch.HelpCompressLong, "Fast Mode is low-latency local compression (live proxy and files). Deep Mode is stronger on-machine LLMLingua: live proxy when Preferences Deep is on, and trim compress --deep for file/batch. Engines are not loaded in Trim cloud.\n\nDeep engines (pick one): v2 LLMLingua-2 (recommended; uses synced Base Model from trim config models), long LongLLMLingua (question-aware; requires --question), v1 LLMLingua (classic).\n\nKeep-rate: --rate or synced compression-rate (Dashboard / trim config rates).\nBase Model (v2 only): Dashboard picker or trim config models / set deep-v2-backbone (billing catalog; default mbert-base, optional xlm-roberta-large). No free-text Hugging Face box.\n\nLicensed use still requires Trim login and network.")
	set(&ch.HelpCompressFlagEngine, "Deep engine: v2 (recommended; uses synced Base Model) | long (needs --question) | v1 (classic). Else prefs deep-engine.")
	ch.ProxyLiveDeepEnabledFmt = "Live Deep Mode on (engine %s, backbone %s, compression_rate %g, oom=%s). Synced from account preferences / billing settings."
	ch.HelpUninstallShort = "Fully remove Trim from this machine (proxy, daemon, data, Deep caches, binary, package install)"
	ch.HelpUninstallLong = "Fully removes Trim from this machine:\n  - stops the local proxy, Cursor Link tunnel, and other Trim processes\n  - removes the OS login daemon\n  - clears CLI credentials\n  - reverts IDE Base URL overrides from trim setup\n  - deletes ~/.trim and ~/.config/trim, Deep caches, sidecars, and the CLI binary (plus Windows user PATH / Apps entry)\n  - when installed via npm / Scoop / Homebrew / winget, schedules that package uninstall after exit (so Windows can release the running binary)\n\nUse --keep-data to skip deleting local data directories, Deep caches, the CLI binary/package, and Windows LocalAppData Trim.\n\nIDE extension is separate: Trim: Clear API Key, then uninstall Trim IDE from the Extensions view.\nIf a detected package install was not auto-removed after exit:\n  brew uninstall trim\n  scoop uninstall trim\n  winget uninstall Trim.CLI\n  npm uninstall -g @usetrim/trim"
	ch.HelpUninstallFlagKeepData = "Keep ~/.trim, ~/.config/trim, Deep Mode caches, the CLI binary/package, and Windows LocalAppData Trim (only stop / unregister Always-on / logout / setup revert)"
	ch.UninstallProcessesStopped = "Stopped other Trim processes on this machine."
	ch.UninstallBinaryScheduledFmt = "Scheduled CLI binary removal after exit: %s"
	ch.UninstallPkgScheduledFmt = "Scheduled package removal after exit: %s"
	ch.UninstallBinaryManualFmt = "Could not schedule binary removal. Delete manually after closing all Trim windows (or reboot if Windows deferred the delete): %s"
	ch.UninstallNextPkgBrew = "Fallback if Homebrew install was not auto-removed: brew uninstall trim"
	ch.UninstallNextPkgScoop = "Fallback if Scoop install was not auto-removed: scoop uninstall trim"
	ch.UninstallNextPkgWinget = "Fallback if winget install was not auto-removed: winget uninstall Trim.CLI"
	ch.UninstallNextPkgNpm = "Fallback if npm install was not auto-removed: npm uninstall -g @usetrim/trim"
	// Offline fail-open purge lists (same defaults as MessageForCode) so uninstall
	// still wipes local data / Deep / sidecars when API chrome is stale or unreachable.
	set(&ch.UninstallHomeDirsRel, ".trim,.config/trim")
	set(&ch.UninstallHfCacheRel, ".cache/huggingface")
	set(&ch.UninstallHfHubSubdir, "hub")
	set(&ch.UninstallHfHubDirs, "models--microsoft--llmlingua-2-bert-base-multilingual-cased-meetingbank,models--microsoft--llmlingua-2-xlm-roberta-large-meetingbank")
	set(&ch.UninstallPipPackages, "llmlingua")
	set(&ch.UninstallPipBins, "pip,pip3")
	set(&ch.UninstallPythonBins, "python,python3")
	set(&ch.UninstallPipModule, "pip")
	set(&ch.UninstallSidecarNames, "trim-deep,trim-deep.exe,optimizer,optimizer.exe,optimizer.py,requirements-deep.txt,requirements.txt")
	set(&ch.UninstallDarwinLogRels, "Library/Logs/trim.log,Library/Logs/trim.err.log")
	set(&ch.UninstallArpDisplayName, "Trim")
	set(&ch.UninstallArpPublisher, "Trim")
	set(&ch.UninstallArpRegKey, "Trim")
	set(&ch.UninstallStarting, "Uninstalling Trim from this machine…")
	set(&ch.UninstallStopped, "Local proxy stop attempted.")
	set(&ch.UninstallDaemonOk, "OS login daemon removed (when supported).")
	set(&ch.UninstallLogoutOk, "CLI credentials cleared.")
	set(&ch.UninstallSetupNone, "No trim setup Base URL overrides found to revert.")
	set(&ch.UninstallSetupRevertedFmt, "Reverted IDE/local proxy Base URL in %s")
	set(&ch.UninstallConfigRemovedFmt, "Removed local data: %s")
	set(&ch.UninstallLogsRemoved, "Removed Trim log files (when present).")
	set(&ch.UninstallDeepOk, "Deep Mode local caches and pip packages cleaned (when present).")
	set(&ch.UninstallBinaryRemovedFmt, "Removed CLI binary: %s")
	set(&ch.UninstallPathRemoved, "Removed Trim install directory from the user PATH (Windows).")
	set(&ch.UninstallArpRemoved, "Removed Trim from Windows Apps & features.")
	set(&ch.UninstallDone, "Trim local uninstall finished.")
	set(&ch.UninstallNextExt, "IDE extension: Trim: Clear API Key (clears Secret Storage + local extension state), then Command Palette → Extensions → uninstall Trim IDE.")
	set(&ch.UninstallSidecarRemovedFmt, "Removed Deep Mode sidecar: %s")
	ch.HelpCursorLinkShort = "Public HTTPS URL to local Trim for Cursor Agent (managed tunnel)"
	ch.HelpCursorLinkLong = "Trim Cursor Link exposes your local Trim proxy through an automatic Cloudflare quick tunnel so Cursor Agent (cloud) can reach it without 127.0.0.1.\n\nCompression, meters, and adapters still run on this PC. The public hostname is only a door. Lifecycle matches trim start / trim stop when enabled.\n\nTypical flow:\n  trim cursor-link enable\n  trim start\n  trim cursor-link url\n  # paste …/cl/<token>/v1 into Cursor Override OpenAI Base URL\n  trim stop\n\nSubcommands: enable, disable, status, url, rotate"
	ch.HelpCursorLinkEnableShort = "Enable Cursor Link (auto tunnel while trim start is running)"
	ch.HelpCursorLinkDisableShort = "Disable Cursor Link and stop any running tunnel"
	ch.HelpCursorLinkStatusShort = "Show Cursor Link preference and live public URL"
	ch.HelpCursorLinkURLShort = "Print the live Cursor Agent Base URL (…/v1) for copy/paste"
	ch.HelpCursorLinkRotateShort = "Rotate the Cursor Link access token (revokes the previous public URL path)"
}
