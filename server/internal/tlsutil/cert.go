// Package tlsutil provides helpers to create and reuse a self-signed
// TLS certificate for the RemoteLauncher server. The package is named
// tlsutil (not tls) because the standard library already owns that
// import path.
package tlsutil

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/hex"
	"encoding/pem"
	"fmt"
	"math/big"
	"net"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const (
	certFileName = "cert.pem"
	keyFileName  = "key.pem"

	dirPerm  os.FileMode = 0o700
	certPerm os.FileMode = 0o644
	keyPerm  os.FileMode = 0o600

	certCommonName   = "remotelauncher"
	certValidFor     = 10 * 365 * 24 * time.Hour
	serialNumberBits = 128
)

// EnsureCert returns paths to a TLS certificate and private key under
// dir. If the files already exist and parse correctly they are reused.
// Otherwise a new self-signed ECDSA P-256 cert is generated with:
//   - CN: remotelauncher
//   - SAN DNS: localhost
//   - SAN IP: 127.0.0.1, ::1, plus every non-loopback unicast IP
//     reported by net.InterfaceAddrs
//   - Validity: 10 years
//   - Key permissions: 0600
//   - Cert permissions: 0644
//
// The directory is created with 0700 if missing.
func EnsureCert(dir string) (certPath, keyPath string, err error) {
	if err := os.MkdirAll(dir, dirPerm); err != nil {
		return "", "", fmt.Errorf("mkdir %s: %w", dir, err)
	}
	certPath = filepath.Join(dir, certFileName)
	keyPath = filepath.Join(dir, keyFileName)

	if _, statCertErr := os.Stat(certPath); statCertErr == nil {
		if _, statKeyErr := os.Stat(keyPath); statKeyErr == nil {
			if _, loadErr := tls.LoadX509KeyPair(certPath, keyPath); loadErr == nil {
				return certPath, keyPath, nil
			}
		}
	}

	if err := generateAndWrite(certPath, keyPath); err != nil {
		return "", "", err
	}
	return certPath, keyPath, nil
}

// generateAndWrite creates a fresh ECDSA P-256 key and self-signed
// certificate and writes them to the given paths with the package's
// canonical permissions.
func generateAndWrite(certPath, keyPath string) error {
	priv, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return fmt.Errorf("generate ecdsa key: %w", err)
	}

	serialLimit := new(big.Int).Lsh(big.NewInt(1), serialNumberBits)
	serial, err := rand.Int(rand.Reader, serialLimit)
	if err != nil {
		return fmt.Errorf("generate serial: %w", err)
	}

	now := time.Now()
	template := &x509.Certificate{
		SerialNumber: serial,
		Subject: pkix.Name{
			CommonName: certCommonName,
		},
		NotBefore:             now,
		NotAfter:              now.Add(certValidFor),
		KeyUsage:              x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
		IsCA:                  false,
		DNSNames:              []string{"localhost"},
		IPAddresses:           collectIPs(),
	}

	derBytes, err := x509.CreateCertificate(rand.Reader, template, template, &priv.PublicKey, priv)
	if err != nil {
		return fmt.Errorf("create certificate: %w", err)
	}

	keyDER, err := x509.MarshalECPrivateKey(priv)
	if err != nil {
		return fmt.Errorf("marshal ec private key: %w", err)
	}

	certPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: derBytes})
	keyPEM := pem.EncodeToMemory(&pem.Block{Type: "EC PRIVATE KEY", Bytes: keyDER})

	if err := writeFileWithPerm(certPath, certPEM, certPerm); err != nil {
		return fmt.Errorf("write certificate: %w", err)
	}
	if err := writeFileWithPerm(keyPath, keyPEM, keyPerm); err != nil {
		return fmt.Errorf("write private key: %w", err)
	}
	return nil
}

// writeFileWithPerm writes data to path and then forces the exact
// requested permissions with Chmod. The explicit Chmod defeats any
// umask that would otherwise trim the key file down from 0600.
func writeFileWithPerm(path string, data []byte, perm os.FileMode) error {
	if err := os.WriteFile(path, data, perm); err != nil {
		return err
	}
	return os.Chmod(path, perm)
}

// collectIPs returns loopback addresses plus every non-loopback unicast
// address found on host network interfaces. The loopback pair is always
// included so that certificate verification succeeds on machines with
// no external interfaces.
func collectIPs() []net.IP {
	ips := []net.IP{
		net.IPv4(127, 0, 0, 1),
		net.IPv6loopback,
	}
	seen := map[string]bool{
		ips[0].String(): true,
		ips[1].String(): true,
	}

	addrs, err := net.InterfaceAddrs()
	if err != nil {
		return ips
	}
	for _, addr := range addrs {
		var ip net.IP
		switch v := addr.(type) {
		case *net.IPNet:
			ip = v.IP
		case *net.IPAddr:
			ip = v.IP
		}
		if ip == nil || ip.IsLoopback() || !ip.IsGlobalUnicast() {
			continue
		}
		key := ip.String()
		if seen[key] {
			continue
		}
		seen[key] = true
		ips = append(ips, ip)
	}
	return ips
}

// Fingerprint returns the SHA-256 fingerprint of the certificate at
// path, formatted as colon-separated hex pairs ("AB:CD:EF:..."). The
// hash is computed over the DER encoding, not the PEM wrapper, to match
// the format produced by `openssl x509 -fingerprint -sha256`.
func Fingerprint(certPath string) (string, error) {
	data, err := os.ReadFile(certPath)
	if err != nil {
		return "", err
	}
	block, _ := pem.Decode(data)
	if block == nil || block.Type != "CERTIFICATE" {
		return "", fmt.Errorf("%s: no CERTIFICATE PEM block", certPath)
	}
	sum := sha256.Sum256(block.Bytes)
	encoded := hex.EncodeToString(sum[:])
	pairs := make([]string, 0, len(sum))
	for i := 0; i < len(encoded); i += 2 {
		pairs = append(pairs, strings.ToUpper(encoded[i:i+2]))
	}
	return strings.Join(pairs, ":"), nil
}
