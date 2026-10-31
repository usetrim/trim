package cli

import (
	"strings"

	"github.com/spf13/cobra"
)

const (
	helpGroupEverydayID = "everyday"
	helpGroupAdvancedID = "advanced"
)

// applyCLIHelpChrome sets cobra Short/Long/flag usage and group titles from
// site_messages-backed chrome. Empty chrome leaves help blank (fail-closed;
// no invent English in the binary).
func applyCLIHelpChrome(ch cliChrome) {
	applyHelpGroups(ch)

	rootCmd.Short = ch.HelpRootShort
	rootCmd.Long = ch.HelpRootLong

	startCmd.Short = ch.HelpStartShort
	startCmd.Long = ch.HelpStartLong
	setFlagUsage(startCmd, "port", ch.HelpStartFlagPort)
	setFlagUsage(startCmd, "foreground", ch.HelpStartFlagForeground)

	stopCmd.Short = ch.HelpStopShort

	loginCmd.Short = ch.HelpLoginShort
	logoutCmd.Short = ch.HelpLogoutShort
	uninstallCmd.Short = ch.HelpUninstallShort
	uninstallCmd.Long = ch.HelpUninstallLong
	setFlagUsage(uninstallCmd, "keep-data", ch.HelpUninstallFlagKeepData)
	statusCmd.Short = ch.HelpStatusShort
	statusCmd.Long = ch.HelpStatusLong
	upgradeCmd.Short = ch.HelpUpgradeShort

	statsCmd.Short = ch.HelpStatsShort
	statsCmd.Long = ch.HelpStatsLong
	setFlagUsage(statsCmd, "tui", ch.HelpStatsFlagTui)

	setupCmd.Short = ch.HelpSetupShort
	setupCmd.Long = ch.HelpSetupLong
	setFlagUsage(setupCmd, "tls", ch.HelpSetupFlagTls)

	versionCmd.Short = ch.HelpVersionShort

	daemonCmd.Short = ch.HelpDaemonShort
	daemonCmd.Long = ch.HelpDaemonLong
	daemonInstallCmd.Short = ch.HelpDaemonInstallShort
	daemonUninstallCmd.Short = ch.HelpDaemonUninstallShort
	daemonStatusCmd.Short = ch.HelpDaemonStatusShort
	daemonRunCmd.Short = ch.HelpDaemonRunShort

	telemetryCmd.Short = ch.HelpTelemetryShort
	telemetryStatusCmd.Short = ch.HelpTelemetryStatusShort
	telemetryDisableCmd.Short = ch.HelpTelemetryDisableShort
	telemetryEnableCmd.Short = ch.HelpTelemetryEnableShort

	autostartCmd.Short = ch.HelpAutostartShort
	autostartCmd.Long = ch.HelpAutostartLong
	autostartStatusCmd.Short = ch.HelpAutostartStatusShort
	autostartEnableCmd.Short = ch.HelpAutostartEnableShort
	autostartDisableCmd.Short = ch.HelpAutostartDisableShort

	tuiCmd.Short = ch.HelpTuiShort
	tuiCmd.Long = ch.HelpTuiLong

	compressCmd.Short = ch.HelpCompressShort
	compressCmd.Long = ch.HelpCompressLong
	setFlagUsage(compressCmd, "mode", ch.HelpCompressFlagMode)
	setFlagUsage(compressCmd, "deep", ch.HelpCompressFlagDeep)
	setFlagUsage(compressCmd, "engine", ch.HelpCompressFlagEngine)
	setFlagUsage(compressCmd, "question", ch.HelpCompressFlagQuestion)
	rateHelp := strings.TrimSpace(ch.HelpCompressFlagRate)
	if rateHelp == "" {
		rateHelp = strings.TrimSpace(ch.HelpCompressFlagTarget)
	}
	setFlagUsage(compressCmd, "rate", rateHelp)
	setFlagUsage(compressCmd, "bootstrap", ch.HelpCompressFlagBootstrap)
	setFlagUsage(compressCmd, "out", ch.HelpCompressFlagOut)

	// Config Short defaults are empty in the binary (fail-closed); apply when present.
	if s := strings.TrimSpace(ch.HelpConfigShort); s != "" {
		configCmd.Short = s
	}
	if s := strings.TrimSpace(ch.HelpConfigLong); s != "" {
		configCmd.Long = s
	}
	if s := strings.TrimSpace(ch.HelpConfigGetShort); s != "" {
		configGetCmd.Short = s
	}
	if s := strings.TrimSpace(ch.HelpConfigSetShort); s != "" {
		configSetCmd.Short = s
	}
	if s := strings.TrimSpace(ch.HelpConfigSyncShort); s != "" {
		configSyncCmd.Short = s
	}
	if s := strings.TrimSpace(ch.HelpConfigRatesShort); s != "" {
		configRatesCmd.Short = s
	}
	if s := strings.TrimSpace(ch.HelpConfigModelsShort); s != "" {
		configModelsCmd.Short = s
	}

	// Cursor Link keeps binary defaults when chrome has not synced these keys yet.
	if s := strings.TrimSpace(ch.HelpCursorLinkShort); s != "" {
		cursorLinkCmd.Short = s
	}
	if s := strings.TrimSpace(ch.HelpCursorLinkLong); s != "" {
		cursorLinkCmd.Long = s
	}
	if s := strings.TrimSpace(ch.HelpCursorLinkEnableShort); s != "" {
		cursorLinkEnableCmd.Short = s
	}
	if s := strings.TrimSpace(ch.HelpCursorLinkDisableShort); s != "" {
		cursorLinkDisableCmd.Short = s
	}
	if s := strings.TrimSpace(ch.HelpCursorLinkStatusShort); s != "" {
		cursorLinkStatusCmd.Short = s
	}
	if s := strings.TrimSpace(ch.HelpCursorLinkURLShort); s != "" {
		cursorLinkURLCmd.Short = s
	}
	if s := strings.TrimSpace(ch.HelpCursorLinkRotateShort); s != "" {
		cursorLinkRotateCmd.Short = s
	}
}

func applyHelpGroups(ch cliChrome) {
	everyday := strings.TrimSpace(ch.HelpGroupEveryday)
	advanced := strings.TrimSpace(ch.HelpGroupAdvanced)
	if everyday == "" || advanced == "" {
		// Fail-closed: no invent group titles - flat command list (order still everyday-first).
		clearHelpGroupIDs()
		return
	}
	for _, g := range rootCmd.Groups() {
		switch g.ID {
		case helpGroupEverydayID:
			g.Title = everyday
		case helpGroupAdvancedID:
			g.Title = advanced
		}
	}
	assignHelpGroupIDs()
}

func clearHelpGroupIDs() {
	for _, c := range []*cobra.Command{
		startCmd, stopCmd, statusCmd, upgradeCmd, autostartCmd, cursorLinkCmd, loginCmd, logoutCmd, uninstallCmd, setupCmd, versionCmd,
		statsCmd, tuiCmd, compressCmd, daemonCmd, telemetryCmd, configCmd,
	} {
		c.GroupID = ""
	}
}

func assignHelpGroupIDs() {
	// Everyday: daily IDE path + local meter (stats) next to cloud status.
	for _, c := range []*cobra.Command{
		startCmd, stopCmd, statusCmd, statsCmd, upgradeCmd, autostartCmd, cursorLinkCmd, loginCmd, logoutCmd, uninstallCmd, setupCmd, versionCmd,
	} {
		c.GroupID = helpGroupEverydayID
	}
	for _, c := range []*cobra.Command{
		tuiCmd, compressCmd, daemonCmd, telemetryCmd, configCmd,
	} {
		c.GroupID = helpGroupAdvancedID
	}
}

func setFlagUsage(cmd *cobra.Command, name, usage string) {
	if f := cmd.Flags().Lookup(name); f != nil {
		f.Usage = usage
	}
}
