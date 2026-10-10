package cli

import (
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/spf13/cobra"
)

var stopCmd = &cobra.Command{
	Use:   "stop",
	Short: "",
	RunE: func(cmd *cobra.Command, args []string) error {
		chrome := loadCLIChrome()
		url := strings.TrimSpace(chrome.ProxyShutdownURL)
		if url == "" {
			if chrome.StopURLMissing != "" {
				return fmt.Errorf("%s", chrome.StopURLMissing)
			}
			return chrome.fail("")
		}
		to, err := requireChromeTimeout(chrome.ProxyShutdownTimeoutMs, "CLI_PROXY_SHUTDOWN_TIMEOUT_MS", chrome)
		if err != nil {
			return err
		}
		client := &http.Client{Timeout: to}
		req, err := http.NewRequest(http.MethodPost, url, nil)
		if err != nil {
			return err
		}
		res, err := client.Do(req)
		if err != nil {
			if chrome.StopFailedFmt != "" {
				return fmt.Errorf(chrome.StopFailedFmt, err.Error())
			}
			return err
		}
		defer res.Body.Close()
		_, _ = io.Copy(io.Discard, io.LimitReader(res.Body, 4096))
		if res.StatusCode != http.StatusOK {
			if chrome.StopFailedFmt != "" {
				return fmt.Errorf(chrome.StopFailedFmt, res.Status)
			}
			return chrome.fail("")
		}
		if chrome.StopOk != "" {
			fmt.Println(chrome.StopOk)
		}
		return nil
	},
}
