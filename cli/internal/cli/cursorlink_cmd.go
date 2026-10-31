package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"github.com/usetrim/trim/cli/internal/cliout"
	"github.com/usetrim/trim/cli/internal/config"
)

var cursorLinkCmd = &cobra.Command{
	Use:     "cursor-link",
	Aliases: []string{"cursorlink", "tunnel"},
	// Short/Long come from site_messages via applyCLIHelpChrome (DB-driven; empty until chrome loads).
	Short: "",
	Long:  "",
}

var cursorLinkEnableCmd = &cobra.Command{
	Use:   "enable",
	Short: "",
	RunE: func(cmd *cobra.Command, args []string) error {
		chrome := loadCLIChrome()
		if err := setCursorLinkEnabled(true); err != nil {
			return err
		}
		if s := strings.TrimSpace(chrome.CursorLinkEnabledOk); s != "" {
			cliout.Success(s)
		}
		if s := strings.TrimSpace(chrome.CursorLinkEnableNext); s != "" {
			cliout.Muted(s)
		}
		if tip := strings.TrimSpace(chrome.StartCursorAgentCloudflaredTip); tip != "" {
			cliout.Muted(tip)
		}
		cfg, err := config.LoadLocal()
		if err == nil {
			port := cursorLinkLocalPortFallback(cfg.Port)
			healthURL := fmt.Sprintf("http://127.0.0.1:%s/health", port)
			if proxyHealthOK(healthURL, 2*time.Second) {
				if url, err := ensureCursorLink(port); err == nil && url != "" {
					lab := strings.TrimSpace(chrome.StartCursorAgentLabel)
					if lab != "" {
						cliout.Table([]cliout.Row{{Label: lab, Value: url, Kind: "accent"}})
					}
				} else if err != nil {
					if fmtStr := strings.TrimSpace(chrome.CursorLinkTunnelNotReadyFmt); fmtStr != "" {
						cliout.Warn(fmt.Sprintf(fmtStr, err))
					}
					if tip := strings.TrimSpace(chrome.CursorLinkTunnelRetry); tip != "" {
						cliout.Muted(tip)
					}
				}
			}
		}
		return nil
	},
}

var cursorLinkDisableCmd = &cobra.Command{
	Use:   "disable",
	Short: "",
	RunE: func(cmd *cobra.Command, args []string) error {
		chrome := loadCLIChrome()
		if err := setCursorLinkEnabled(false); err != nil {
			return err
		}
		if s := strings.TrimSpace(chrome.CursorLinkDisabledOk); s != "" {
			cliout.Success(s)
		}
		if s := strings.TrimSpace(chrome.CursorLinkDisabledMuted); s != "" {
			cliout.Muted(s)
		}
		return nil
	},
}

var cursorLinkStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "",
	RunE: func(cmd *cobra.Command, args []string) error {
		chrome := loadCLIChrome()
		st, err := loadCursorLinkState()
		if err != nil {
			return err
		}
		if flagJSON {
			return json.NewEncoder(os.Stdout).Encode(st)
		}
		enabled := strings.TrimSpace(chrome.CursorLinkStateOff)
		if st.Enabled {
			enabled = strings.TrimSpace(chrome.CursorLinkStateOn)
		}
		rows := []cliout.Row{}
		if lab := strings.TrimSpace(chrome.CursorLinkLabelState); lab != "" && enabled != "" {
			rows = append(rows, cliout.Row{Label: lab, Value: enabled, Kind: "accent"})
		}
		if u := cursorLinkPublicBaseURL(st); u != "" {
			if lab := strings.TrimSpace(chrome.CursorLinkLabelPublicURL); lab != "" {
				rows = append(rows, cliout.Row{Label: lab, Value: u, Kind: "ok"})
			}
		} else if st.Enabled {
			if lab := strings.TrimSpace(chrome.CursorLinkLabelPublicURL); lab != "" {
				pending := strings.TrimSpace(chrome.CursorLinkPublicURLPending)
				if pending != "" {
					rows = append(rows, cliout.Row{Label: lab, Value: pending, Kind: "warn"})
				}
			}
		}
		if st.LocalPort != "" {
			if lab := strings.TrimSpace(chrome.CursorLinkLabelLocalPort); lab != "" {
				rows = append(rows, cliout.Row{Label: lab, Value: st.LocalPort, Kind: ""})
			}
		}
		if st.PID > 0 {
			if lab := strings.TrimSpace(chrome.CursorLinkLabelTunnelPID); lab != "" {
				rows = append(rows, cliout.Row{Label: lab, Value: formatPID(st.PID), Kind: ""})
			}
		}
		if len(rows) > 0 {
			cliout.Table(rows)
		}
		if st.Enabled {
			if s := strings.TrimSpace(chrome.CursorLinkStatusPaste); s != "" {
				cliout.Muted(s)
			}
			if s := strings.TrimSpace(chrome.CursorLinkStatusPrivacy); s != "" {
				cliout.Muted(s)
			}
		} else if s := strings.TrimSpace(chrome.CursorLinkStatusEnableHint); s != "" {
			cliout.Muted(s)
		}
		return nil
	},
}

var cursorLinkRotateCmd = &cobra.Command{
	Use:   "rotate",
	Short: "",
	RunE: func(cmd *cobra.Command, args []string) error {
		chrome := loadCLIChrome()
		st, err := loadCursorLinkState()
		if err != nil {
			return err
		}
		if !st.Enabled {
			msg := strings.TrimSpace(chrome.CursorLinkDisabledErr)
			if msg == "" {
				return chrome.fail("")
			}
			return fmt.Errorf("%s", msg)
		}
		if _, err := rotateCursorLinkAccessToken(); err != nil {
			return err
		}
		if s := strings.TrimSpace(chrome.CursorLinkRotatedOk); s != "" {
			cliout.Success(s)
		}
		if s := strings.TrimSpace(chrome.CursorLinkRotateMuted); s != "" {
			cliout.Muted(s)
		}
		st, _ = loadCursorLinkState()
		if u := cursorLinkPublicBaseURL(st); u != "" {
			lab := strings.TrimSpace(chrome.CursorLinkLabelNewURL)
			if lab != "" {
				cliout.Table([]cliout.Row{{Label: lab, Value: u, Kind: "accent"}})
			}
		}
		return nil
	},
}

var cursorLinkURLCmd = &cobra.Command{
	Use:   "url",
	Short: "",
	RunE: func(cmd *cobra.Command, args []string) error {
		chrome := loadCLIChrome()
		st, err := loadCursorLinkState()
		if err != nil {
			return err
		}
		if !st.Enabled {
			msg := strings.TrimSpace(chrome.CursorLinkDisabledStartErr)
			if msg == "" {
				return chrome.fail("")
			}
			return fmt.Errorf("%s", msg)
		}
		u := cursorLinkPublicBaseURL(st)
		if u == "" {
			cfg, _ := config.LoadLocal()
			port := cursorLinkLocalPortFallback("")
			if cfg.Port != "" {
				port = cursorLinkLocalPortFallback(cfg.Port)
			}
			ensureURL, ensureErr := ensureCursorLink(port)
			u = strings.TrimSpace(ensureURL)
			if u == "" {
				fmtStr := strings.TrimSpace(chrome.CursorLinkNoURLFmt)
				if fmtStr == "" {
					return chrome.fail("")
				}
				return fmt.Errorf(fmtStr, ensureErr)
			}
		}
		if flagJSON {
			return json.NewEncoder(os.Stdout).Encode(map[string]string{"public_base_url": u})
		}
		fmt.Println(u)
		return nil
	},
}

func init() {
	cursorLinkCmd.AddCommand(cursorLinkEnableCmd)
	cursorLinkCmd.AddCommand(cursorLinkDisableCmd)
	cursorLinkCmd.AddCommand(cursorLinkStatusCmd)
	cursorLinkCmd.AddCommand(cursorLinkURLCmd)
	cursorLinkCmd.AddCommand(cursorLinkRotateCmd)
}

// maybeStartCursorLinkAfterProxy is called when the local proxy is healthy.
func maybeStartCursorLinkAfterProxy(listenPort string) {
	st, err := loadCursorLinkState()
	if err != nil || !st.Enabled {
		return
	}
	chrome := loadCLIChrome()
	url, err := ensureCursorLink(listenPort)
	if err != nil {
		if fmtStr := strings.TrimSpace(chrome.CursorLinkWarnFmt); fmtStr != "" {
			cliout.Warn(fmt.Sprintf(fmtStr, err))
		}
		if tip := strings.TrimSpace(chrome.StartCursorAgentCloudflaredTip); tip != "" {
			cliout.Muted(tip)
		}
		return
	}
	if strings.TrimSpace(url) == "" {
		return
	}
	lab := strings.TrimSpace(chrome.StartCursorAgentLabel)
	if lab != "" {
		cliout.Table([]cliout.Row{
			{Label: lab, Value: url, Kind: "accent"},
		})
	}
	if tip := strings.TrimSpace(chrome.StartCursorAgentTipOn); tip != "" {
		cliout.Muted(tip)
	}
	if tipFmt := strings.TrimSpace(chrome.StartCursorAgentLocalTip); tipFmt != "" {
		cliout.Muted(fmt.Sprintf(tipFmt, strings.TrimSpace(listenPort)))
	}
}
