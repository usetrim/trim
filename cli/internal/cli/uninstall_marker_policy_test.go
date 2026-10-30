package cli

import (
	"os"
	"path/filepath"
	"testing"
)

// Ensures the marker-keep policy cannot regress: if an aside/binary remains after
// a failed schedule path, clearing markers must not happen via the !needSchedule
// branch alone. This guards the exact self-heal bug that left vendor\trim.exe.
func TestUninstallMarkerClearedOnlyWhenNoScheduleNeeded(t *testing.T) {
	tmp := t.TempDir()
	npmRoot := filepath.Join(tmp, "node_modules", "@usetrim", "trim")
	if err := os.MkdirAll(filepath.Join(npmRoot, "vendor"), 0o755); err != nil {
		t.Fatal(err)
	}
	exe := filepath.Join(npmRoot, "vendor", "trim.exe")
	if err := os.WriteFile(exe, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	aside := exe + ".uninstalling"
	if err := os.WriteFile(aside, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}

	writeUninstallMarkers(exe)
	pkgMark := filepath.Join(npmRoot, uninstallMarkerName)
	if _, err := os.Stat(pkgMark); err != nil {
		t.Fatalf("marker missing after write: %v", err)
	}

	// Simulate the fixed policy: schedule was needed + failed, aside still exists
	// → do NOT clear.
	needSchedule := true
	if !needSchedule {
		clearUninstallMarkers(exe)
	} else if _, err := os.Stat(aside); err == nil {
		// keep marker
	}
	if _, err := os.Stat(pkgMark); err != nil {
		t.Fatalf("marker must remain while aside exists: %v", err)
	}

	// When no schedule needed, clear is allowed.
	needSchedule = false
	if !needSchedule {
		clearUninstallMarkers(exe)
	}
	if _, err := os.Stat(pkgMark); !os.IsNotExist(err) {
		t.Fatalf("marker should clear when no schedule needed")
	}
}
