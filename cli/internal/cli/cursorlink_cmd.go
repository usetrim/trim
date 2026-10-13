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
	Short:   "Trim Cursor Link - public HTTPS URL to local Trim (for Cursor Agent)",
	Long: `Trim Cursor Link exposes your local Trim proxy through an automatic Cloudflare
quick tunnel so Cursor Agent (cloud) can reach it without using 127.0.0.1.

Compression, meters, and adapters still run on this PC. The public hostname is
only a door. Lifecycle matches trim start / trim stop when enabled.

Typical flow:
  trim cursor-link enable
  trim start
  trim cursor-link url          # paste …/v1 into Cursor Override OpenAI Base URL
  trim stop                     # stops Trim and the tunnel together
`,
}

var cursorLinkEnableCmd = &cobra.Command{
	Use:   "enable",
	Short: "Enable Cursor Link (auto tunnel while trim start is running)",
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := setCursorLinkEnabled(true); err != nil {
			return err
		}
		cliout.Success("Cursor Link enabled")
		cliout.Muted("Next: trim start  →  then trim cursor-link url  →  paste into Cursor Override OpenAI Base URL (…/v1)")
		cliout.Muted("cloudflared: Trim auto-downloads to ~/.trim/bin on enable/start when missing (all OS). Optional brew/winget only if installed. Or set TRIM_CLOUDFLARED. Local Trim still does the work.")
		cfg, err := config.LoadLocal()
		if err == nil {
			port := cursorLinkLocalPortFallback(cfg.Port)
			healthURL := fmt.Sprintf("http://127.0.0.1:%s/health", port)
			if proxyHealthOK(healthURL, 2*time.Second) {
				if url, err := ensureCursorLink(port); err == nil && url != "" {
					cliout.Table([]cliout.Row{{Label: "Cursor Agent Base URL", Value: url, Kind: "accent"}})
				} else if err != nil {
					cliout.Warn(fmt.Sprintf("Trim is up but tunnel not ready yet: %v", err))
					cliout.Muted("Run trim start again (Trim auto-downloads cloudflared when needed), or: trim cursor-link status")
				}
			}
		}
		return nil
	},
}

var cursorLinkDisableCmd = &cobra.Command{
	Use:   "disable",
	Short: "Disable Cursor Link and stop any running tunnel",
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := setCursorLinkEnabled(false); err != nil {
			return err
		}
		cliout.Success("Cursor Link disabled (tunnel stopped if it was running)")
		cliout.Muted("Cursor Agent will not get a public Trim URL until you enable again. Local 127.0.0.1 still works for on-device clients.")
		return nil
	},
}

var cursorLinkStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show Cursor Link preference and live public URL",
	RunE: func(cmd *cobra.Command, args []string) error {
		st, err := loadCursorLinkState()
		if err != nil {
			return err
		}
		if flagJSON {
			return json.NewEncoder(os.Stdout).Encode(st)
		}
		enabled := "off"
		if st.Enabled {
			enabled = "on"
		}
		rows := []cliout.Row{
			{Label: "Cursor Link", Value: enabled, Kind: "accent"},
		}
		if u := cursorLinkPublicBaseURL(st); u != "" {
			rows = append(rows, cliout.Row{Label: "Public Base URL", Value: u, Kind: "ok"})
		} else if st.Enabled {
			rows = append(rows, cliout.Row{Label: "Public Base URL", Value: "(not up - run trim start)", Kind: "warn"})
		}
		if st.LocalPort != "" {
			rows = append(rows, cliout.Row{Label: "Local port", Value: st.LocalPort, Kind: ""})
		}
		if st.PID > 0 {
			rows = append(rows, cliout.Row{Label: "Tunnel PID", Value: formatPID(st.PID), Kind: ""})
		}
		cliout.Table(rows)
		if st.Enabled {
			cliout.Muted("Paste Public Base URL into Cursor → Settings → Models → Override OpenAI Base URL")
			cliout.Muted("Privacy: URL is public; Fast/Deep compression stays on this PC")
		} else {
			cliout.Muted("Enable with: trim cursor-link enable")
		}
		return nil
	},
}

var cursorLinkRotateCmd = &cobra.Command{
	Use:   "rotate",
	Short: "Rotate the Cursor Link access token (revokes the previous public URL path)",
	RunE: func(cmd *cobra.Command, args []string) error {
		st, err := loadCursorLinkState()
		if err != nil {
			return err
		}
		if !st.Enabled {
			return fmt.Errorf("Cursor Link is disabled - run: trim cursor-link enable")
		}
		if _, err := rotateCursorLinkAccessToken(); err != nil {
			return err
		}
		cliout.Success("Cursor Link token rotated")
		cliout.Muted("Old public path is revoked immediately (Trim reads the new token live). Update Cursor Override OpenAI Base URL.")
		st, _ = loadCursorLinkState()
		if u := cursorLinkPublicBaseURL(st); u != "" {
			cliout.Table([]cliout.Row{{Label: "New Cursor Agent Base URL", Value: u, Kind: "accent"}})
		}
		return nil
	},
}

var cursorLinkURLCmd = &cobra.Command{
	Use:   "url",
	Short: "Print the live Cursor Agent Base URL (…/v1) for copy/paste",
	RunE: func(cmd *cobra.Command, args []string) error {
		st, err := loadCursorLinkState()
		if err != nil {
			return err
		}
		if !st.Enabled {
			return fmt.Errorf("Cursor Link is disabled - run: trim cursor-link enable && trim start")
		}
		u := cursorLinkPublicBaseURL(st)
		if u == "" {
			cfg, _ := config.LoadLocal()
			port := cursorLinkLocalPortFallback("")
			if cfg.Port != "" {
				port = cursorLinkLocalPortFallback(cfg.Port)
			}
			var ensureErr error
			u, ensureErr = ensureCursorLink(port)
			if ensureErr != nil {
				return fmt.Errorf("no public URL yet (%v) - is Trim running? trim start", ensureErr)
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
	url, err := ensureCursorLink(listenPort)
	if err != nil {
		cliout.Warn(fmt.Sprintf("Cursor Link: %v", err))
		cliout.Muted("Trim auto-downloads cloudflared to ~/.trim/bin when missing (all OS). Optional brew/winget only if installed. Or set TRIM_CLOUDFLARED, then: trim cursor-link status")
		return
	}
	if strings.TrimSpace(url) == "" {
		return
	}
	cliout.Table([]cliout.Row{
		{Label: "Cursor Agent Base URL", Value: url, Kind: "accent"},
	})
	cliout.Muted("Cursor Agent: paste that URL (not 127.0.0.1) - avoids \"private networks forbidden\"")
	cliout.Muted("Local tools / Claude Code: keep using http://127.0.0.1:" + strings.TrimSpace(listenPort) + " as documented")
}
