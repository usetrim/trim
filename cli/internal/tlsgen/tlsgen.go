package tlsgen

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"math/big"
	"net"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// Options for GenerateLocal. Org and ValidityDays are required (no invent defaults).
type Options struct {
	Organization string
	ValidityDays int
}

// GenerateLocal writes a self-signed cert + key for localhost HTTPS IDE overrides.
// Files: cert.pem, key.pem under dir. Returns absolute paths.
func GenerateLocal(dir string, opts Options) (certPath, keyPath string, err error) {
	org := strings.TrimSpace(opts.Organization)
	if org == "" {
		return "", "", fmt.Errorf("CONFIG_ENV_TRIM_TLS_ORG_REQUIRED")
	}
	if opts.ValidityDays < 1 || opts.ValidityDays > 825 {
		return "", "", fmt.Errorf("CONFIG_ENV_TRIM_TLS_VALIDITY_DAYS_INVALID")
	}
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return "", "", err
	}
	certPath = filepath.Join(dir, "cert.pem")
	keyPath = filepath.Join(dir, "key.pem")

	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return "", "", err
	}

	serial, err := rand.Int(rand.Reader, new(big.Int).Lsh(big.NewInt(1), 128))
	if err != nil {
		return "", "", err
	}

	tmpl := &x509.Certificate{
		SerialNumber: serial,
		Subject: pkix.Name{
			Organization: []string{org},
			CommonName:   "localhost",
		},
		NotBefore:             time.Now().Add(-time.Hour),
		NotAfter:              time.Now().Add(time.Duration(opts.ValidityDays) * 24 * time.Hour),
		KeyUsage:              x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
		DNSNames:              []string{"localhost"},
		IPAddresses:           []net.IP{net.ParseIP("127.0.0.1"), net.ParseIP("::1")},
	}

	der, err := x509.CreateCertificate(rand.Reader, tmpl, tmpl, &key.PublicKey, key)
	if err != nil {
		return "", "", err
	}

	certOut, err := os.OpenFile(certPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o600)
	if err != nil {
		return "", "", err
	}
	if err := pem.Encode(certOut, &pem.Block{Type: "CERTIFICATE", Bytes: der}); err != nil {
		_ = certOut.Close()
		return "", "", err
	}
	_ = certOut.Close()

	keyBytes, err := x509.MarshalECPrivateKey(key)
	if err != nil {
		return "", "", err
	}
	keyOut, err := os.OpenFile(keyPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o600)
	if err != nil {
		return "", "", err
	}
	if err := pem.Encode(keyOut, &pem.Block{Type: "EC PRIVATE KEY", Bytes: keyBytes}); err != nil {
		_ = keyOut.Close()
		return "", "", err
	}
	_ = keyOut.Close()

	return certPath, keyPath, nil
}

// ConfigDir returns ~/.config/trim/tls (or %APPDATA%/trim/tls on Windows).
func ConfigDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".config", "trim", "tls"), nil
}
