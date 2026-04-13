package tlsutil

import (
	"crypto/tls"
	"crypto/x509"
	"encoding/pem"
	"net"
	"os"
	"path/filepath"
	"regexp"
	"testing"
	"time"
)

func TestEnsureCert_GeneratesNewFiles(t *testing.T) {
	// Use a sub-directory so EnsureCert has to create it itself: the
	// caller's t.TempDir() would already exist with the runtime's
	// default 0755 permissions and MkdirAll leaves existing dirs alone.
	dir := filepath.Join(t.TempDir(), "remotelauncher")
	certPath, keyPath, err := EnsureCert(dir)
	if err != nil {
		t.Fatalf("EnsureCert: %v", err)
	}
	if _, err := os.Stat(certPath); err != nil {
		t.Fatalf("stat cert: %v", err)
	}
	if _, err := os.Stat(keyPath); err != nil {
		t.Fatalf("stat key: %v", err)
	}

	if _, err := tls.LoadX509KeyPair(certPath, keyPath); err != nil {
		t.Fatalf("LoadX509KeyPair: %v", err)
	}

	info, err := os.Stat(keyPath)
	if err != nil {
		t.Fatalf("stat key: %v", err)
	}
	if perm := info.Mode().Perm(); perm != 0o600 {
		t.Errorf("key perm = %#o, want 0600", perm)
	}

	certInfo, err := os.Stat(certPath)
	if err != nil {
		t.Fatalf("stat cert: %v", err)
	}
	if perm := certInfo.Mode().Perm(); perm != 0o644 {
		t.Errorf("cert perm = %#o, want 0644", perm)
	}

	dirInfo, err := os.Stat(dir)
	if err != nil {
		t.Fatalf("stat dir: %v", err)
	}
	if perm := dirInfo.Mode().Perm(); perm != 0o700 {
		t.Errorf("dir perm = %#o, want 0700", perm)
	}
}

func TestEnsureCert_ReusesExistingFiles(t *testing.T) {
	dir := t.TempDir()
	certPath, keyPath, err := EnsureCert(dir)
	if err != nil {
		t.Fatalf("first EnsureCert: %v", err)
	}

	firstCert, err := os.Stat(certPath)
	if err != nil {
		t.Fatalf("stat cert: %v", err)
	}
	firstKey, err := os.Stat(keyPath)
	if err != nil {
		t.Fatalf("stat key: %v", err)
	}

	// Sleep a bit so any rewrite would shift mtime observably.
	time.Sleep(20 * time.Millisecond)

	certPath2, keyPath2, err := EnsureCert(dir)
	if err != nil {
		t.Fatalf("second EnsureCert: %v", err)
	}
	if certPath2 != certPath || keyPath2 != keyPath {
		t.Errorf("paths changed between calls: %q/%q vs %q/%q", certPath, keyPath, certPath2, keyPath2)
	}

	secondCert, err := os.Stat(certPath)
	if err != nil {
		t.Fatalf("stat cert (second): %v", err)
	}
	secondKey, err := os.Stat(keyPath)
	if err != nil {
		t.Fatalf("stat key (second): %v", err)
	}

	if !firstCert.ModTime().Equal(secondCert.ModTime()) {
		t.Errorf("cert mtime changed on reuse: %v -> %v", firstCert.ModTime(), secondCert.ModTime())
	}
	if !firstKey.ModTime().Equal(secondKey.ModTime()) {
		t.Errorf("key mtime changed on reuse: %v -> %v", firstKey.ModTime(), secondKey.ModTime())
	}
}

func TestEnsureCert_RegeneratesCorruptedFiles(t *testing.T) {
	dir := t.TempDir()
	certPath := filepath.Join(dir, "cert.pem")
	keyPath := filepath.Join(dir, "key.pem")
	if err := os.MkdirAll(dir, 0o700); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(certPath, []byte("not a pem"), 0o644); err != nil {
		t.Fatalf("write cert junk: %v", err)
	}
	if err := os.WriteFile(keyPath, []byte("also not a pem"), 0o600); err != nil {
		t.Fatalf("write key junk: %v", err)
	}

	if _, _, err := EnsureCert(dir); err != nil {
		t.Fatalf("EnsureCert on corrupted files: %v", err)
	}
	if _, err := tls.LoadX509KeyPair(certPath, keyPath); err != nil {
		t.Fatalf("LoadX509KeyPair after regeneration: %v", err)
	}
}

func TestEnsureCert_CreatesDirectoryIfMissing(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "sub", "dir")
	if _, err := os.Stat(dir); !os.IsNotExist(err) {
		t.Fatalf("precondition: dir must not exist, got err=%v", err)
	}
	certPath, keyPath, err := EnsureCert(dir)
	if err != nil {
		t.Fatalf("EnsureCert: %v", err)
	}
	if _, err := os.Stat(certPath); err != nil {
		t.Errorf("stat cert: %v", err)
	}
	if _, err := os.Stat(keyPath); err != nil {
		t.Errorf("stat key: %v", err)
	}
}

func TestEnsureCert_CertContainsExpectedSANs(t *testing.T) {
	dir := t.TempDir()
	certPath, _, err := EnsureCert(dir)
	if err != nil {
		t.Fatalf("EnsureCert: %v", err)
	}

	data, err := os.ReadFile(certPath)
	if err != nil {
		t.Fatalf("read cert: %v", err)
	}
	block, _ := pem.Decode(data)
	if block == nil {
		t.Fatalf("decode pem: nil")
	}
	cert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		t.Fatalf("parse cert: %v", err)
	}

	if cert.Subject.CommonName != "remotelauncher" {
		t.Errorf("CN = %q, want remotelauncher", cert.Subject.CommonName)
	}
	if cert.PublicKeyAlgorithm != x509.ECDSA {
		t.Errorf("PublicKeyAlgorithm = %v, want ECDSA", cert.PublicKeyAlgorithm)
	}

	hasLocalhost := false
	for _, name := range cert.DNSNames {
		if name == "localhost" {
			hasLocalhost = true
			break
		}
	}
	if !hasLocalhost {
		t.Errorf("DNSNames = %v, want to contain localhost", cert.DNSNames)
	}

	has4, has6 := false, false
	for _, ip := range cert.IPAddresses {
		if ip.Equal(net.IPv4(127, 0, 0, 1)) {
			has4 = true
		}
		if ip.Equal(net.IPv6loopback) {
			has6 = true
		}
	}
	if !has4 {
		t.Errorf("IPAddresses missing 127.0.0.1: %v", cert.IPAddresses)
	}
	if !has6 {
		t.Errorf("IPAddresses missing ::1: %v", cert.IPAddresses)
	}
	if len(cert.IPAddresses) < 2 {
		t.Errorf("len(IPAddresses) = %d, want >= 2", len(cert.IPAddresses))
	}

	validity := cert.NotAfter.Sub(cert.NotBefore)
	if validity < 9*365*24*time.Hour {
		t.Errorf("validity = %v, want >= ~10 years", validity)
	}

	if cert.KeyUsage&x509.KeyUsageDigitalSignature == 0 {
		t.Errorf("KeyUsage missing DigitalSignature: %v", cert.KeyUsage)
	}
	hasServerAuth := false
	for _, eku := range cert.ExtKeyUsage {
		if eku == x509.ExtKeyUsageServerAuth {
			hasServerAuth = true
			break
		}
	}
	if !hasServerAuth {
		t.Errorf("ExtKeyUsage missing ServerAuth: %v", cert.ExtKeyUsage)
	}
	if cert.IsCA {
		t.Error("IsCA = true, want false")
	}
}

func TestFingerprint_Format(t *testing.T) {
	dir := t.TempDir()
	certPath, _, err := EnsureCert(dir)
	if err != nil {
		t.Fatalf("EnsureCert: %v", err)
	}
	fp, err := Fingerprint(certPath)
	if err != nil {
		t.Fatalf("Fingerprint: %v", err)
	}
	if len(fp) != 95 {
		t.Errorf("len(fp) = %d, want 95", len(fp))
	}
	re := regexp.MustCompile(`^([0-9A-F]{2}:){31}[0-9A-F]{2}$`)
	if !re.MatchString(fp) {
		t.Errorf("fp = %q, not uppercase colon-separated hex", fp)
	}
}

func TestFingerprint_Stable(t *testing.T) {
	dir := t.TempDir()
	certPath, _, err := EnsureCert(dir)
	if err != nil {
		t.Fatalf("EnsureCert: %v", err)
	}
	fp1, err := Fingerprint(certPath)
	if err != nil {
		t.Fatalf("first Fingerprint: %v", err)
	}
	fp2, err := Fingerprint(certPath)
	if err != nil {
		t.Fatalf("second Fingerprint: %v", err)
	}
	if fp1 != fp2 {
		t.Errorf("fingerprint not stable: %q vs %q", fp1, fp2)
	}
}

func TestFingerprint_NotFound(t *testing.T) {
	if _, err := Fingerprint(filepath.Join(t.TempDir(), "does-not-exist.pem")); err == nil {
		t.Error("Fingerprint on missing file: nil error, want error")
	}
}

func TestFingerprint_NotPEM(t *testing.T) {
	path := filepath.Join(t.TempDir(), "junk.pem")
	if err := os.WriteFile(path, []byte("this is not pem"), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}
	if _, err := Fingerprint(path); err == nil {
		t.Error("Fingerprint on non-pem: nil error, want error")
	}
}

func TestEnsureCert_ReadOnlyDirectory(t *testing.T) {
	if os.Getuid() == 0 {
		t.Skip("running as root bypasses directory permissions")
	}
	dir := filepath.Join(t.TempDir(), "readonly")
	if err := os.Mkdir(dir, 0o500); err != nil {
		t.Fatalf("mkdir ro: %v", err)
	}
	t.Cleanup(func() { _ = os.Chmod(dir, 0o700) })
	if _, _, err := EnsureCert(dir); err == nil {
		t.Error("EnsureCert on read-only dir: nil error, want error")
	}
}

func TestCollectIPs_HasLoopback(t *testing.T) {
	ips := collectIPs()
	if len(ips) < 2 {
		t.Fatalf("len(collectIPs) = %d, want >= 2", len(ips))
	}
	has4, has6 := false, false
	for _, ip := range ips {
		if ip.Equal(net.IPv4(127, 0, 0, 1)) {
			has4 = true
		}
		if ip.Equal(net.IPv6loopback) {
			has6 = true
		}
	}
	if !has4 {
		t.Errorf("collectIPs missing 127.0.0.1: %v", ips)
	}
	if !has6 {
		t.Errorf("collectIPs missing ::1: %v", ips)
	}
}
