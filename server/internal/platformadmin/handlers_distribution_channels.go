package platformadmin

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// syncGitHubRepoTraffic writes clones + views for a repo under the given distribution source.
// Uses TRIM_GITHUB_TOKEN (needs traffic:read on the repo). Fail-closed on HTTP errors.
func (h *Handler) syncGitHubRepoTraffic(ctx context.Context, source, repo string) error {
	token := strings.TrimSpace(h.Config.GitHubToken)
	repo = strings.TrimSpace(repo)
	source = strings.ToLower(strings.TrimSpace(source))
	if token == "" {
		return fmt.Errorf("ADMIN_GITHUB_NOT_CONFIGURED")
	}
	if repo == "" || source == "" {
		return fmt.Errorf("ADMIN_DIST_CHANNEL_REPO_REQUIRED")
	}
	client := &http.Client{Timeout: 30 * time.Second}

	type dayCount struct {
		Timestamp string `json:"timestamp"`
		Count     int64  `json:"count"`
	}
	fetchSeries := func(path, metric string) error {
		req, err := http.NewRequestWithContext(ctx, http.MethodGet,
			"https://api.github.com/repos/"+repo+"/"+path, nil)
		if err != nil {
			return err
		}
		req.Header.Set("Authorization", "Bearer "+token)
		req.Header.Set("Accept", "application/vnd.github+json")
		res, err := client.Do(req)
		if err != nil {
			return err
		}
		defer res.Body.Close()
		body, _ := io.ReadAll(res.Body)
		if res.StatusCode >= 300 {
			return fmt.Errorf("ADMIN_DISTRIBUTION_SYNC_FAILED")
		}
		var envelope map[string]json.RawMessage
		if err := json.Unmarshal(body, &envelope); err != nil {
			return err
		}
		raw, ok := envelope[metric]
		if !ok {
			// clones payload key is "clones"; views is "views"
			return fmt.Errorf("ADMIN_DISTRIBUTION_SYNC_FAILED")
		}
		var series []dayCount
		if err := json.Unmarshal(raw, &series); err != nil {
			return err
		}
		for _, row := range series {
			day := strings.TrimSpace(row.Timestamp)
			if len(day) >= 10 {
				day = day[:10]
			}
			if day == "" {
				continue
			}
			_, _ = h.DB.Exec(ctx, `
				insert into public.distribution_daily_stats (day, source, metric, country, path, value)
				values ($1::date, $2, $3, '', '', $4)
				on conflict (day, source, metric, country, path) do update set value = excluded.value
			`, day, source, metric, row.Count)
		}
		return nil
	}

	if err := fetchSeries("traffic/clones", "clones"); err != nil {
		return err
	}
	if err := fetchSeries("traffic/views", "views"); err != nil {
		return err
	}
	_, _ = h.DB.Exec(ctx, `
		insert into public.distribution_sync_state (source, last_synced_at, last_error, meta)
		values ($1, now(), null, jsonb_build_object('repo', $2::text))
		on conflict (source) do update set last_synced_at = now(), last_error = null, meta = excluded.meta
	`, source, repo)
	return nil
}

// syncVSCodeMarketplace writes today's install count for a public Marketplace extension.
// extensionID is publisher.name (e.g. usetrim.trim-ide). No invent publisher.
func (h *Handler) syncVSCodeMarketplace(ctx context.Context, extensionID string) error {
	extensionID = strings.TrimSpace(extensionID)
	if extensionID == "" || !strings.Contains(extensionID, ".") {
		return fmt.Errorf("ADMIN_DIST_MARKETPLACE_ID_REQUIRED")
	}
	payload := map[string]any{
		"filters": []map[string]any{
			{
				"criteria": []map[string]any{
					{"filterType": 7, "value": extensionID},
				},
				"pageNumber": 1,
				"pageSize":   1,
			},
		},
		// IncludeStatistics | IncludeVersions | IncludeFiles (public gallery flags).
		"flags": 914,
	}
	raw, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		"https://marketplace.visualstudio.com/_apis/public/gallery/extensionquery?api-version=7.1-preview.1",
		bytes.NewReader(raw))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json;api-version=7.1-preview.1")
	client := &http.Client{Timeout: 30 * time.Second}
	res, err := client.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	body, _ := io.ReadAll(res.Body)
	if res.StatusCode >= 300 {
		return fmt.Errorf("ADMIN_DISTRIBUTION_SYNC_FAILED")
	}
	var parsed struct {
		Results []struct {
			Extensions []struct {
				Statistics []struct {
					StatisticName string  `json:"statisticName"`
					Value         float64 `json:"value"`
				} `json:"statistics"`
			} `json:"extensions"`
		} `json:"results"`
	}
	if err := json.Unmarshal(body, &parsed); err != nil {
		return err
	}
	var installs int64
	found := false
	for _, r := range parsed.Results {
		for _, ext := range r.Extensions {
			for _, st := range ext.Statistics {
				name := strings.ToLower(strings.TrimSpace(st.StatisticName))
				if name == "install" || name == "installcount" || name == "downloadcount" {
					installs = int64(st.Value)
					found = true
				}
			}
		}
	}
	if !found {
		return fmt.Errorf("ADMIN_DIST_MARKETPLACE_STATS_MISSING")
	}
	day := time.Now().UTC().Format("2006-01-02")
	_, err = h.DB.Exec(ctx, `
		insert into public.distribution_daily_stats (day, source, metric, country, path, value)
		values ($1::date, 'marketplace', 'installs', '', '', $2)
		on conflict (day, source, metric, country, path) do update set value = excluded.value
	`, day, installs)
	if err != nil {
		return err
	}
	_, _ = h.DB.Exec(ctx, `
		insert into public.distribution_sync_state (source, last_synced_at, last_error, meta)
		values ('marketplace', now(), null, jsonb_build_object('extension_id', $1::text, 'installs', $2::bigint))
		on conflict (source) do update set last_synced_at = now(), last_error = null, meta = excluded.meta
	`, extensionID, installs)
	return nil
}

func (h *Handler) recordDistSyncError(ctx context.Context, source, errMsg string) {
	source = strings.ToLower(strings.TrimSpace(source))
	if source == "" || errMsg == "" {
		return
	}
	_, _ = h.DB.Exec(ctx, `
		insert into public.distribution_sync_state (source, last_synced_at, last_error, meta)
		values ($1, now(), $2, '{}'::jsonb)
		on conflict (source) do update set last_synced_at = now(), last_error = excluded.last_error
	`, source, errMsg)
}

func (h *Handler) distChannelNotice(err error) string {
	if err == nil {
		return ""
	}
	msg := err.Error()
	if strings.HasPrefix(msg, "ADMIN_") {
		return h.msg(msg)
	}
	return h.msg("ADMIN_DISTRIBUTION_SYNC_FAILED")
}
