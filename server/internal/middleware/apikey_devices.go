package middleware

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"
)

var errHardwareDeviceLimit = errors.New("hardware device limit")
var errAPIKeyMaxDevicesMissing = errors.New("api key max devices missing")
var errHardwareUnbound = errors.New("api key has no registered devices")

// enforceAPIKeyDeviceAllowlist requires every API key to be device-bound and
// the request X-Hardware-UUID to match the allowlist.
// Bound = rows in api_key_devices and/or legacy api_keys.hardware_uuid.
// Unbound keys (dashboard-issued with no hardware) are rejected until a device
// is registered via JWT account APIs - stolen keys must not work from any PC.
// Unknown hardware is rejected (no auto-enroll).
func (m *AuthQuota) enforceAPIKeyDeviceAllowlist(
	ctx context.Context,
	keyID, legacyHW, requestHW, _agentID string,
) error {
	requestHW = strings.TrimSpace(requestHW)
	legacyHW = strings.TrimSpace(legacyHW)

	devices, err := m.listAPIKeyDevices(ctx, keyID)
	if err != nil {
		return err
	}
	if legacyHW != "" {
		found := false
		for _, d := range devices {
			if strings.EqualFold(d, legacyHW) {
				found = true
				break
			}
		}
		if !found {
			devices = append(devices, legacyHW)
		}
	}

	if len(devices) == 0 {
		return errHardwareUnbound
	}

	if requestHW == "" {
		return errHardwareBoundRequired
	}
	for _, d := range devices {
		if strings.EqualFold(d, requestHW) {
			_ = m.touchAPIKeyDevice(ctx, keyID, requestHW)
			return nil
		}
	}
	return errHardwareMismatch
}

func (m *AuthQuota) listAPIKeyDevices(ctx context.Context, keyID string) ([]string, error) {
	if m.DB == nil {
		return nil, fmt.Errorf("database unavailable")
	}
	rows, err := m.DB.Query(ctx, `
		select hardware_uuid from public.api_key_devices where key_id = $1::uuid
	`, keyID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]string, 0, 4)
	for rows.Next() {
		var hw string
		if rows.Scan(&hw) != nil {
			continue
		}
		hw = strings.TrimSpace(hw)
		if hw != "" {
			out = append(out, hw)
		}
	}
	return out, rows.Err()
}

func (m *AuthQuota) touchAPIKeyDevice(ctx context.Context, keyID, hw string) error {
	_, err := m.DB.Exec(ctx, `
		update public.api_key_devices
		set last_seen_at = now()
		where key_id = $1::uuid and hardware_uuid = $2
	`, keyID, hw)
	return err
}

func (m *AuthQuota) apiKeyMaxDevices(ctx context.Context) (int, error) {
	if m.DB == nil {
		return 0, fmt.Errorf("database unavailable")
	}
	var n *int
	err := m.DB.QueryRow(ctx, `
		select api_key_max_devices from public.admin_product_settings where id = 'default'
	`).Scan(&n)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return 0, errAPIKeyMaxDevicesMissing
		}
		return 0, err
	}
	if n == nil || *n < 1 {
		return 0, errAPIKeyMaxDevicesMissing
	}
	return *n, nil
}

// APIKeyMaxDevices is exported for account handlers (JWT device registration).
func (m *AuthQuota) APIKeyMaxDevices(ctx context.Context) (int, error) {
	return m.apiKeyMaxDevices(ctx)
}
