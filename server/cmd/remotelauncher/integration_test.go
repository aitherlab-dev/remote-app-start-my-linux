//go:build integration
// +build integration

package main_test

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"io"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"syscall"
	"testing"
	"time"
)

func bytesContainsAny(b []byte, needles ...string) bool {
	for _, n := range needles {
		if bytes.Contains(b, []byte(n)) {
			return true
		}
	}
	return false
}

const (
	serverAddr      = "127.0.0.1:8443"
	serverURL       = "https://" + serverAddr
	startupDeadline = 5 * time.Second
	shutdownWindow  = 15 * time.Second
)

// insecureTLSClient returns an http.Client that skips TLS verification,
// which is the only way to reach the integration server's self-signed
// certificate from a test. Never use this pattern in production code.
func insecureTLSClient() *http.Client {
	return &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true}, //nolint:gosec // test-only trust
		},
		Timeout: 5 * time.Second,
	}
}

func TestServer_EndToEnd(t *testing.T) {
	tmp := t.TempDir()

	xdgHome := filepath.Join(tmp, "xdghome")
	appsDir := filepath.Join(xdgHome, "applications")
	if err := os.MkdirAll(appsDir, 0o755); err != nil {
		t.Fatalf("mkdir applications: %v", err)
	}
	writeDesktop(t, appsDir, "alpha.desktop", "Alpha", "alpha")
	writeDesktop(t, appsDir, "beta.desktop", "Beta", "beta")

	// XDG_CONFIG_HOME holds the server's self-signed TLS material so
	// the test never touches the developer's real ~/.config.
	xdgConfig := filepath.Join(tmp, "xdgconfig")
	if err := os.MkdirAll(xdgConfig, 0o700); err != nil {
		t.Fatalf("mkdir xdgconfig: %v", err)
	}

	binary := filepath.Join(tmp, "remotelauncher")
	build := exec.Command("go", "build", "-o", binary, "./cmd/remotelauncher")
	build.Dir = repoRoot(t)
	build.Stdout = os.Stdout
	build.Stderr = os.Stderr
	if err := build.Run(); err != nil {
		t.Fatalf("build: %v", err)
	}

	// Isolate the child process from the host XDG dirs so the app
	// count is deterministic (two, from our fixture). XDG_CONFIG_HOME
	// redirects the TLS cert into our tmp dir.
	env := append(os.Environ(),
		"XDG_DATA_HOME="+xdgHome,
		"XDG_DATA_DIRS="+xdgHome,
		"XDG_CONFIG_HOME="+xdgConfig,
		"HOME="+xdgHome,
	)

	cmd := exec.Command(binary)
	cmd.Env = env
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Start(); err != nil {
		t.Fatalf("start server: %v", err)
	}
	t.Cleanup(func() {
		if cmd.Process != nil {
			_ = cmd.Process.Kill()
		}
	})

	if err := waitForPort(serverAddr, startupDeadline); err != nil {
		t.Fatalf("wait for port: %v", err)
	}

	client := insecureTLSClient()

	// GET /api/status
	{
		body := httpGetOK(t, client, serverURL+"/api/status")
		var status struct {
			Version         string `json:"version"`
			AppsCount       int    `json:"apps_count"`
			CertFingerprint string `json:"cert_fingerprint"`
		}
		if err := json.Unmarshal(body, &status); err != nil {
			t.Fatalf("decode status: %v (body=%s)", err, body)
		}
		if status.Version == "" {
			t.Errorf("status.Version empty")
		}
		if status.AppsCount < 0 {
			t.Errorf("status.AppsCount = %d, want >= 0", status.AppsCount)
		}
		re := regexp.MustCompile(`^([0-9A-F]{2}:){31}[0-9A-F]{2}$`)
		if !re.MatchString(status.CertFingerprint) {
			t.Errorf("cert_fingerprint = %q, not a 32-pair uppercase hex string", status.CertFingerprint)
		}
		// The cert should be under xdgConfig/remotelauncher.
		if _, err := os.Stat(filepath.Join(xdgConfig, "remotelauncher", "cert.pem")); err != nil {
			t.Errorf("cert.pem missing under XDG_CONFIG_HOME: %v", err)
		}
	}

	// GET /api/apps — DefaultPaths hardcodes /usr/share/applications
	// so the child process also sees the host's system entries. We
	// can't isolate that without a config flag (comes in S5.1), so
	// the assertion is: our two fixtures must be present and no
	// Exec-like field may leak.
	{
		body := httpGetOK(t, client, serverURL+"/api/apps")
		if bytesContainsAny(body, "\"exec\"", "\"Exec\"", "\"tryexec\"", "\"TryExec\"") {
			t.Errorf("/api/apps leaked an exec field: %s", body)
		}
		var list []map[string]any
		if err := json.Unmarshal(body, &list); err != nil {
			t.Fatalf("decode apps: %v (body=%s)", err, body)
		}
		if len(list) < 2 {
			t.Fatalf("len(apps) = %d, want >= 2", len(list))
		}
		seen := map[string]bool{}
		for _, item := range list {
			if id, ok := item["id"].(string); ok {
				seen[id] = true
			}
		}
		for _, want := range []string{"alpha", "beta"} {
			if !seen[want] {
				t.Errorf("fixture %q missing from /api/apps (XDG_DATA_HOME override not applied?)", want)
			}
		}
	}

	// GET /api/nonexistent -> 404 JSON
	{
		resp, err := client.Get(serverURL + "/api/nonexistent")
		if err != nil {
			t.Fatalf("GET nonexistent: %v", err)
		}
		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		if resp.StatusCode != http.StatusNotFound {
			t.Errorf("status = %d, want 404", resp.StatusCode)
		}
		var errBody struct {
			Error struct {
				Code string `json:"code"`
			} `json:"error"`
		}
		if err := json.Unmarshal(body, &errBody); err != nil {
			t.Fatalf("decode error body: %v (body=%s)", err, body)
		}
		if errBody.Error.Code == "" {
			t.Errorf("error.code empty, body=%s", body)
		}
	}

	// Graceful shutdown on SIGTERM.
	if err := cmd.Process.Signal(syscall.SIGTERM); err != nil {
		t.Fatalf("signal SIGTERM: %v", err)
	}

	done := make(chan error, 1)
	go func() { done <- cmd.Wait() }()
	select {
	case err := <-done:
		if err != nil {
			t.Errorf("process exit error: %v", err)
		}
	case <-time.After(shutdownWindow):
		t.Errorf("server did not exit within %s", shutdownWindow)
		_ = cmd.Process.Kill()
	}
}

// writeDesktop drops a minimal Application entry into dir.
func writeDesktop(t *testing.T, dir, file, name, exec string) {
	t.Helper()
	content := "[Desktop Entry]\nType=Application\nName=" + name + "\nExec=/bin/" + exec + "\n"
	if err := os.WriteFile(filepath.Join(dir, file), []byte(content), 0o644); err != nil {
		t.Fatalf("write %s: %v", file, err)
	}
}

// waitForPort dials addr repeatedly until it accepts or the deadline
// expires. It's used to avoid racing the child process boot-up.
func waitForPort(addr string, within time.Duration) error {
	deadline := time.Now().Add(within)
	for {
		conn, err := net.DialTimeout("tcp", addr, 200*time.Millisecond)
		if err == nil {
			conn.Close()
			return nil
		}
		if time.Now().After(deadline) {
			return err
		}
		time.Sleep(50 * time.Millisecond)
	}
}

func httpGetOK(t *testing.T, client *http.Client, url string) []byte {
	t.Helper()
	resp, err := client.Get(url)
	if err != nil {
		t.Fatalf("GET %s: %v", url, err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("read body %s: %v", url, err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("GET %s status = %d, body=%s", url, resp.StatusCode, body)
	}
	return body
}

// repoRoot returns the server/ directory (the module root) so the
// `go build ./cmd/remotelauncher` invocation resolves the package.
func repoRoot(t *testing.T) string {
	t.Helper()
	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	// tests run from server/cmd/remotelauncher — walk up two levels.
	return filepath.Clean(filepath.Join(wd, "..", ".."))
}
