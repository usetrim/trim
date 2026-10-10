package cli

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "trim",
	Short: "",
	Long:  "",
}

func Execute() error {
	// Warm help before first --help when possible (API may be offline: blank help).
	applyCLIHelpChrome(loadCLIChrome())
	return rootCmd.Execute()
}

func init() {
	// Preserve AddCommand order (everyday: start/stop first; not alpha).
	cobra.EnableCommandSorting = false

	// Group titles come from site_messages (CLI_HELP_GROUP_*); empty until chrome loads.
	rootCmd.AddGroup(
		&cobra.Group{ID: helpGroupEverydayID, Title: ""},
		&cobra.Group{ID: helpGroupAdvancedID, Title: ""},
	)

	// Set PersistentPreRun in init to avoid package init cycle with applyCLIHelpChrome.
	rootCmd.PersistentPreRun = func(cmd *cobra.Command, args []string) {
		applyCLIHelpChrome(loadCLIChrome())
	}

	// Everyday first (order within group), then advanced.
	// stats sits with status: cloud quota vs local Deep/savings meter.
	rootCmd.AddCommand(startCmd)
	rootCmd.AddCommand(stopCmd)
	rootCmd.AddCommand(statusCmd)
	rootCmd.AddCommand(statsCmd)
	rootCmd.AddCommand(upgradeCmd)
	rootCmd.AddCommand(autostartCmd)
	rootCmd.AddCommand(loginCmd)
	rootCmd.AddCommand(logoutCmd)
	rootCmd.AddCommand(uninstallCmd)
	rootCmd.AddCommand(setupCmd)
	rootCmd.AddCommand(versionCmd)

	rootCmd.AddCommand(tuiCmd)
	rootCmd.AddCommand(compressCmd)
	rootCmd.AddCommand(daemonCmd)
	rootCmd.AddCommand(telemetryCmd)
	rootCmd.AddCommand(configCmd)

	assignHelpGroupIDs()
}

func printErr(err error) {
	chrome := loadCLIChrome()
	if chrome.ErrFmt != "" {
		fmt.Fprintf(os.Stderr, chrome.ErrFmt, err)
		return
	}
	// Fail-closed: no invent English prefix when chrome unavailable.
	fmt.Fprintf(os.Stderr, "%v\n", err)
}

// MainErrFmt returns site_messages-backed stderr format for cmd/trim (CLI_MAIN_ERR_FMT).
// Empty when API chrome is unavailable (fail-closed; caller must not invent a prefix).
func MainErrFmt() string {
	return loadCLIChrome().MainErrFmt
}
