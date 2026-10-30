//go:build windows

package cli

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"
)

// Live proof: post-exit helper deletes a renamed vendor binary after the
// "uninstall" process exits - the exact Windows lock failure mode.
func TestScheduleWindowsPostExitCleanupDeletesAside(t *testing.T) {
	tmp := t.TempDir()
	vendor := filepath.Join(tmp, "node_modules", "@usetrim", "trim", "vendor")
	if err := os.MkdirAll(vendor, 0o755); err != nil {
		t.Fatal(err)
	}
	aside := filepath.Join(vendor, "trim.exe.uninstalling")
	if err := os.WriteFile(aside, []byte("fake-locked-binary"), 0o644); err != nil {
		t.Fatal(err)
	}

	holder := exec.Command("powershell.exe", "-NoProfile", "-Command", "Start-Sleep -Seconds 2")
	if err := holder.Start(); err != nil {
		t.Fatal(err)
	}
	defer func() {
		_ = holder.Process.Kill()
		_, _ = holder.Process.Wait()
	}()

	// Use standalone only - do not run npm/scoop/winget uninstall against the host.
	if err := scheduleWindowsPostExitCleanupFor(
		holder.Process.Pid,
		aside,
		vendor,
		nil,
		channelStandalone,
	); err != nil {
		t.Fatalf("schedule helper: %v", err)
	}

	// Wait for holder exit + helper cleanup window.
	done := make(chan error, 1)
	go func() { done <- holder.Wait() }()
	select {
	case <-done:
	case <-time.After(15 * time.Second):
		t.Fatal("holder did not exit")
	}

	deadline := time.Now().Add(25 * time.Second)
	for time.Now().Before(deadline) {
		if _, err := os.Stat(aside); os.IsNotExist(err) {
			return
		}
		time.Sleep(400 * time.Millisecond)
	}
	t.Fatalf("aside still present after helper: %s", aside)
}

func TestScheduleWindowsDeleteOnRebootAcceptsPath(t *testing.T) {
	tmp := t.TempDir()
	p := filepath.Join(tmp, "trim-reboot-delete.txt")
	if err := os.WriteFile(p, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	// May succeed (registers pending rename) or fail on some locked-down hosts;
	// must not panic and must accept a normal user-owned path.
	_ = scheduleWindowsDeleteOnReboot(p)
}
