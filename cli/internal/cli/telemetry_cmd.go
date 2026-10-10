package cli

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/usetrim/trim/cli/internal/telemetry"
)

var telemetryCmd = &cobra.Command{
	Use:   "telemetry",
	Short: "",
}

var telemetryStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "",
	RunE: func(cmd *cobra.Command, args []string) error {
		chrome := loadCLIChrome()
		if telemetry.IsEnabled() {
			if chrome.TelemetryEnabled != "" {
				fmt.Println(chrome.TelemetryEnabled)
			}
			if chrome.TelemetryDisableHint != "" {
				fmt.Println(chrome.TelemetryDisableHint)
			}
			if chrome.TelemetryEnvHint != "" {
				fmt.Println(chrome.TelemetryEnvHint)
			}
			return nil
		}
		if chrome.TelemetryDisabled != "" {
			fmt.Println(chrome.TelemetryDisabled)
		}
		if chrome.TelemetryEnableHint != "" {
			fmt.Println(chrome.TelemetryEnableHint)
		}
		return nil
	},
}

var telemetryDisableCmd = &cobra.Command{
	Use:   "disable",
	Short: "",
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := telemetry.Disable(); err != nil {
			return err
		}
		chrome := loadCLIChrome()
		if chrome.TelemetryDisabledOk != "" {
			fmt.Println(chrome.TelemetryDisabledOk)
		}
		return nil
	},
}

var telemetryEnableCmd = &cobra.Command{
	Use:   "enable",
	Short: "",
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := telemetry.Enable(); err != nil {
			return err
		}
		chrome := loadCLIChrome()
		if chrome.TelemetryEnabledOk != "" {
			fmt.Println(chrome.TelemetryEnabledOk)
		}
		return nil
	},
}

func init() {
	telemetryCmd.AddCommand(telemetryStatusCmd)
	telemetryCmd.AddCommand(telemetryDisableCmd)
	telemetryCmd.AddCommand(telemetryEnableCmd)
}
