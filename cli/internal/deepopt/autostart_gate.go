package deepopt

import (
	"os"
	"path/filepath"
)

// AutostartBlocked reports enterprise/privacy managed-off for everyday proxy auto-start.
// Mirrors telemetry DO_NOT_TRACK-style gates; does not invent enable when blocked.
func AutostartBlocked() (blocked bool, reason string) {
	if os.Getenv("DO_NOT_TRACK") == "1" {
		return true, "dnt"
	}
	if os.Getenv("TRIM_AUTOSTART_DISABLED") == "1" {
		return true, "managed"
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return false, ""
	}
	path := filepath.Join(home, ".config", "trim", "autostart.off")
	if _, err := os.Stat(path); err == nil {
		return true, "managed"
	}
	return false, ""
}

// WriteAutostartOffMarker creates ~/.config/trim/autostart.off (local managed off).
func WriteAutostartOffMarker() error {
	home, err := os.UserHomeDir()
	if err != nil {
		return err
	}
	dir := filepath.Join(home, ".config", "trim")
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(dir, "autostart.off"), []byte("disabled\n"), 0o600)
}

// ClearAutostartOffMarker removes the local managed-off file.
func ClearAutostartOffMarker() error {
	home, err := os.UserHomeDir()
	if err != nil {
		return err
	}
	path := filepath.Join(home, ".config", "trim", "autostart.off")
	err = os.Remove(path)
	if err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}
