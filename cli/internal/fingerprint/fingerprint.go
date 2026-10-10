package fingerprint

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"runtime"
	"strings"

	"github.com/denisbrodbeck/machineid"
)

// HardwareUUID returns a stable SHA256 machine fingerprint for CLI auth binding.
func HardwareUUID() (string, error) {
	id, err := machineid.ProtectedID("TrimCLI")
	if err != nil {
		hostname, _ := os.Hostname()
		id = fmt.Sprintf("%s-%s-%s", hostname, runtime.GOOS, runtime.GOARCH)
	}
	sum := sha256.Sum256([]byte(strings.TrimSpace(id)))
	return hex.EncodeToString(sum[:]), nil
}
