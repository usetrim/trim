package cli

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/usetrim/trim/server/pkg/pow"
)

// PoW UX chrome from site_messages (set when CLI chrome is loaded).
var (
	powClientRequired string
	pow428Missing     string
	powSolveFailed    string
	powParseFailedFmt string
)

func powMsg(cached, code string) string {
	if s := strings.TrimSpace(cached); s != "" {
		return s
	}
	return code
}

// doCLIRequest sends an authenticated CLI request. On HTTP 428 with a PoW
// challenge (free tier), it solves once and retries with X-Trim-PoW.
// client must be non-nil with an ops-configured timeout (no invent).
func doCLIRequest(client *http.Client, req *http.Request, token, hmacSecret string, body []byte) (*http.Response, error) {
	if client == nil {
		return nil, fmt.Errorf("%s", powMsg(powClientRequired, "CLI_POW_CLIENT_REQUIRED"))
	}
	if err := attachCLIHeaders(req, token, hmacSecret, body); err != nil {
		return nil, err
	}
	if body != nil {
		req.Body = io.NopCloser(bytes.NewReader(body))
		req.ContentLength = int64(len(body))
	}

	res, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	if res.StatusCode != http.StatusPreconditionRequired {
		return res, nil
	}

	challengeRaw := res.Header.Get(pow.ChallengeHeader)
	_ = res.Body.Close()
	if challengeRaw == "" {
		return nil, fmt.Errorf("%s", powMsg(pow428Missing, "CLI_POW_428_MISSING"))
	}
	ch, err := pow.ParseChallenge(challengeRaw)
	if err != nil {
		fmtStr := powMsg(powParseFailedFmt, "CLI_POW_PARSE_FAILED_FMT")
		return nil, fmt.Errorf(fmtStr, err)
	}
	nonce := pow.Solve(ch)
	if nonce == "" {
		return nil, fmt.Errorf("%s", powMsg(powSolveFailed, "CLI_POW_SOLVE_FAILED"))
	}

	retry, err := http.NewRequest(req.Method, req.URL.String(), nil)
	if err != nil {
		return nil, err
	}
	if err := attachCLIHeaders(retry, token, hmacSecret, body); err != nil {
		return nil, err
	}
	retry.Header.Set(pow.HeaderName, nonce)
	if body != nil {
		retry.Body = io.NopCloser(bytes.NewReader(body))
		retry.ContentLength = int64(len(body))
	}
	return client.Do(retry)
}
