package middleware

import (
	"bufio"
	"fmt"
	"net"
	"os"
	"strings"
	"sync"

	"github.com/oschwald/maxminddb-golang"
)

// Offline IP risk sources (no third-party HTTP). Prefer these over ipwho.is when present.
//
// Supported inputs for TRIM_GEOLITE_*_MMDB_PATH:
//   - Raw MaxMind GeoLite2 / GeoIP2 .mmdb (Anonymous-IP or ASN)
//   - Sidecar CIDR export next to an .mmdb: <file>.mmdb.cidr.txt
//   - Plain text: CIDR lines ("1.2.3.0/24"), ASN lines ("AS9009" / "9009")
//   - Comments: lines starting with # are ignored

var (
	geoMu     sync.RWMutex
	geoNets   []*net.IPNet
	geoASNs   map[string]struct{}
	geoAnonDB *maxminddb.Reader
	geoASNDB  *maxminddb.Reader
	geoReady  bool
)

type anonIPRecord struct {
	IsAnonymous        bool `maxminddb:"is_anonymous"`
	IsAnonymousVPN     bool `maxminddb:"is_anonymous_vpn"`
	IsHostingProvider  bool `maxminddb:"is_hosting_provider"`
	IsPublicProxy      bool `maxminddb:"is_public_proxy"`
	IsResidentialProxy bool `maxminddb:"is_residential_proxy"`
	IsTorExitNode      bool `maxminddb:"is_tor_exit_node"`
}

type asnRecord struct {
	AutonomousSystemNumber uint   `maxminddb:"autonomous_system_number"`
	AutonomousSystemOrg    string `maxminddb:"autonomous_system_organization"`
}

// InitGeoLite loads optional offline blocklists and/or raw MaxMind MMDB readers.
func InitGeoLite(asnPath, anonymousPath string) error {
	geoMu.Lock()
	defer geoMu.Unlock()

	closeGeoReadersLocked()
	geoNets = nil
	geoASNs = map[string]struct{}{}
	geoReady = false

	var firstErr error
	for _, path := range []string{asnPath, anonymousPath} {
		path = strings.TrimSpace(path)
		if path == "" {
			continue
		}
		if err := loadGeoPathLocked(path); err != nil && firstErr == nil {
			firstErr = err
		}
	}

	if geoAnonDB != nil || geoASNDB != nil || len(geoNets) > 0 || len(geoASNs) > 0 {
		geoReady = true
		return nil
	}
	if asnPath == "" && anonymousPath == "" {
		return nil
	}
	if firstErr != nil {
		return firstErr
	}
	return fmt.Errorf("geolite paths set but no usable MMDB or CIDR/ASN data loaded")
}

func closeGeoReadersLocked() {
	if geoAnonDB != nil {
		_ = geoAnonDB.Close()
		geoAnonDB = nil
	}
	if geoASNDB != nil {
		_ = geoASNDB.Close()
		geoASNDB = nil
	}
}

func loadGeoPathLocked(path string) error {
	lower := strings.ToLower(path)
	if strings.HasSuffix(lower, ".mmdb") {
		if err := openMMDBLocked(path); err == nil {
			return nil
		} else {
			// Prefer raw MMDB; fall back to sidecar CIDR export if present.
			if sidecar := resolveSidecarCIDR(path); sidecar != "" {
				return loadRiskFileLocked(sidecar)
			}
			return err
		}
	}
	resolved := resolveRiskFile(path)
	if resolved == "" {
		return fmt.Errorf("ip risk file not found: %s", path)
	}
	return loadRiskFileLocked(resolved)
}

func openMMDBLocked(path string) error {
	db, err := maxminddb.Open(path)
	if err != nil {
		return fmt.Errorf("open MMDB %s: %w", path, err)
	}

	metaType := strings.ToLower(db.Metadata.DatabaseType)
	switch {
	case strings.Contains(metaType, "anonymous"):
		if geoAnonDB != nil {
			_ = geoAnonDB.Close()
		}
		geoAnonDB = db
		return nil
	case strings.Contains(metaType, "asn"):
		if geoASNDB != nil {
			_ = geoASNDB.Close()
		}
		geoASNDB = db
		return nil
	default:
		// Unknown MMDB type: keep as anonymous-style lookup (is_* flags) if decode works.
		if geoAnonDB != nil {
			_ = geoAnonDB.Close()
		}
		geoAnonDB = db
		return nil
	}
}

func resolveSidecarCIDR(mmdbPath string) string {
	sidecar := mmdbPath + ".cidr.txt"
	if st, err := os.Stat(sidecar); err == nil && !st.IsDir() {
		return sidecar
	}
	alt := strings.TrimSuffix(mmdbPath, ".mmdb") + ".cidr.txt"
	if st, err := os.Stat(alt); err == nil && !st.IsDir() {
		return alt
	}
	return ""
}

func resolveRiskFile(path string) string {
	if strings.HasSuffix(strings.ToLower(path), ".mmdb") {
		return resolveSidecarCIDR(path)
	}
	if st, err := os.Stat(path); err == nil && !st.IsDir() {
		return path
	}
	return ""
}

func loadRiskFileLocked(path string) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()

	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) == 0 {
			continue
		}
		token := fields[0]
		upper := strings.ToUpper(token)
		if strings.HasPrefix(upper, "AS") || isAllDigits(token) {
			asn := strings.TrimPrefix(upper, "AS")
			if asn != "" {
				geoASNs[asn] = struct{}{}
			}
			continue
		}
		if !strings.Contains(token, "/") {
			token = token + "/32"
			if strings.Contains(token, ":") {
				token = strings.TrimSuffix(token, "/32") + "/128"
			}
		}
		_, network, err := net.ParseCIDR(token)
		if err != nil {
			continue
		}
		geoNets = append(geoNets, network)
	}
	return sc.Err()
}

func isAllDigits(s string) bool {
	if s == "" {
		return false
	}
	for _, c := range s {
		if c < '0' || c > '9' {
			return false
		}
	}
	return true
}

// lookupGeoLiteRisk returns (risky, resolved). resolved=false means no offline data.
func lookupGeoLiteRisk(ip string) (risky bool, resolved bool) {
	geoMu.RLock()
	defer geoMu.RUnlock()
	if !geoReady {
		return false, false
	}
	parsed := net.ParseIP(ip)
	if parsed == nil {
		return false, false
	}

	if geoAnonDB != nil {
		var rec anonIPRecord
		if err := geoAnonDB.Lookup(parsed, &rec); err == nil {
			risky := rec.IsAnonymous ||
				rec.IsAnonymousVPN ||
				rec.IsPublicProxy ||
				rec.IsResidentialProxy ||
				rec.IsTorExitNode ||
				rec.IsHostingProvider
			return risky, true
		}
	}

	if geoASNDB != nil {
		var rec asnRecord
		if err := geoASNDB.Lookup(parsed, &rec); err == nil && rec.AutonomousSystemNumber > 0 {
			asn := fmt.Sprintf("%d", rec.AutonomousSystemNumber)
			if _, blocked := geoASNs[asn]; blocked {
				return true, true
			}
			// ASN DB alone without a blocklist: resolved but not risky by default.
			if len(geoASNs) > 0 {
				return false, true
			}
			return false, true
		}
	}

	for _, n := range geoNets {
		if n.Contains(parsed) {
			return true, true
		}
	}
	if len(geoNets) > 0 {
		return false, true
	}
	return false, false
}

// asnBlockedOffline reports whether an ASN string is in the offline ASN set.
func asnBlockedOffline(asn string) bool {
	geoMu.RLock()
	defer geoMu.RUnlock()
	if !geoReady || len(geoASNs) == 0 {
		return false
	}
	asn = strings.TrimPrefix(strings.ToUpper(strings.TrimSpace(asn)), "AS")
	_, ok := geoASNs[asn]
	return ok
}
