package cli

import (
	"github.com/spf13/cobra"
	"github.com/usetrim/trim/cli/internal/config"
	"github.com/usetrim/trim/cli/internal/storage"
	"github.com/usetrim/trim/cli/internal/tui"
)

var tuiCmd = &cobra.Command{
	Use:   "tui",
	Short: "",
	Long:  "",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.LoadLocal()
		if err != nil {
			return err
		}
		token, _ := storage.GetToken()
		chrome := fetchLocalChrome(cfg)
		return tui.Run(cfg.Port, cfg.APIBaseURL, token, cfg.CLIHMACSecret, Version, cfg.SavingsUsdPerMTok, cfg.CLIHTTPTimeoutSec, cfg.TUIRefreshSec, cfg.TUISeriesDays, cfg.TUIErrorMaxChars, chrome)
	},
}
