package paddleapi

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

type apiErrorBody struct {
	Error *struct {
		Code   string `json:"code"`
		Detail string `json:"detail"`
	} `json:"error"`
}

func (c *Client) doJSON(ctx context.Context, method, path string, reqBody any, out any) error {
	if c == nil {
		return fmt.Errorf("paddle client is nil")
	}
	var reader io.Reader
	if reqBody != nil {
		payload, err := json.Marshal(reqBody)
		if err != nil {
			return err
		}
		reader = bytes.NewReader(payload)
	}
	req, err := http.NewRequestWithContext(ctx, method, c.BaseURL+path, reader)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+c.APIKey)
	req.Header.Set("Paddle-Version", "1")
	if reqBody != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	res, err := c.HTTP.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	raw, _ := io.ReadAll(res.Body)

	if out != nil && len(raw) > 0 {
		if err := json.Unmarshal(raw, out); err != nil {
			return fmt.Errorf("paddle decode %s %s: %w body=%s", method, path, err, string(raw))
		}
	}
	if res.StatusCode >= 400 {
		detail := strings.TrimSpace(string(raw))
		code := ""
		var errBody apiErrorBody
		if json.Unmarshal(raw, &errBody) == nil && errBody.Error != nil {
			code = strings.TrimSpace(errBody.Error.Code)
			detail = strings.TrimSpace(errBody.Error.Detail)
			if detail == "" {
				detail = code
			}
		}
		// Keep status + code so callers can detect not_found even when detail is prose
		// ("Product not found.") without the underscore form of the error code.
		if code != "" {
			return fmt.Errorf("paddle api %s %s: status=%d code=%s detail=%s", method, path, res.StatusCode, code, detail)
		}
		return fmt.Errorf("paddle api %s %s: status=%d detail=%s", method, path, res.StatusCode, detail)
	}
	return nil
}
