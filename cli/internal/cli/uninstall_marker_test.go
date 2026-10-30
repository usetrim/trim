package cli

import (
	"os"
	"path/filepath"
	"testing"
)

func TestWriteAndClearUninstallMarkers(t *testing.T) {
	tmp := t.TempDir()
	npmRoot := filepath.Join(tmp, "node_modules", "@usetrim", "trim")
	if err := os.MkdirAll(filepath.Join(npmRoot, "vendor"), 0o755); err != nil {
		t.Fatal(err)
	}
	exe := filepath.Join(npmRoot, "vendor", "trim.exe")
	if err := os.WriteFile(exe, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}

	// Seed a legacy TEMP marker; clear must remove it without write recreating it.
	tempMark := filepath.Join(os.TempDir(), "trim-uninstalling")
	if err := os.WriteFile(tempMark, []byte("1"), 0o600); err != nil {
		t.Fatal(err)
	}

	writeUninstallMarkers(exe)
	pkgMark := filepath.Join(npmRoot, uninstallMarkerName)
	if _, err := os.Stat(pkgMark); err != nil {
		t.Fatalf("pkg marker missing: %v", err)
	}
	// Package-local only - write must not create/refresh TEMP.
	if _, err := os.Stat(tempMark); err != nil {
		t.Fatalf("legacy temp marker should still exist until clear: %v", err)
	}

	clearUninstallMarkers(exe)
	if _, err := os.Stat(tempMark); !os.IsNotExist(err) {
		t.Fatalf("temp marker should be cleared")
	}
	if _, err := os.Stat(pkgMark); !os.IsNotExist(err) {
		t.Fatalf("pkg marker should be cleared")
	}
}
