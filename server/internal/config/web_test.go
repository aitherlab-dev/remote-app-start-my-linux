package config

import (
	"strings"
	"testing"
)

func TestValidate_WebListenAddrLoopback(t *testing.T) {
	cases := []struct {
		name    string
		addr    string
		wantErr string
	}{
		{"127.0.0.1 ok", "127.0.0.1:17843", ""},
		{"::1 ok", "[::1]:17843", ""},
		{"localhost ok", "localhost:17843", ""},
		{"127.0.0.2 ok (loopback range)", "127.0.0.2:17843", ""},
		{"empty", "", "must not be empty"},
		{"bind all", ":17843", "loopback"},
		{"lan", "192.168.1.10:17843", "loopback"},
		{"public", "1.2.3.4:17843", "loopback"},
		{"no port", "127.0.0.1", "missing port"},
		{"junk host", "not-a-host:17843", "loopback"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			c := Defaults()
			c.Web.Enabled = true
			c.Web.ListenAddr = tc.addr
			err := c.Validate()
			if tc.wantErr == "" {
				if err != nil {
					t.Errorf("Validate() = %v, want nil", err)
				}
				return
			}
			if err == nil {
				t.Fatalf("Validate() = nil, want error containing %q", tc.wantErr)
			}
			if !strings.Contains(err.Error(), tc.wantErr) {
				t.Errorf("Validate() error = %q, want contains %q", err, tc.wantErr)
			}
		})
	}
}

func TestValidate_WebDisabledSkipsAddrCheck(t *testing.T) {
	c := Defaults()
	c.Web.Enabled = false
	c.Web.ListenAddr = "0.0.0.0:17843" // would fail loopback check if enabled
	if err := c.Validate(); err != nil {
		t.Errorf("Validate() = %v, want nil (web disabled)", err)
	}
}

func TestLoadFile_WebSection(t *testing.T) {
	path := writeTempFile(t, `
[web]
enabled = false
listen_addr = "127.0.0.1:19999"
`)

	c := Defaults()
	if err := LoadFile(&c, path); err != nil {
		t.Fatalf("LoadFile: %v", err)
	}
	if c.Web.Enabled {
		t.Error("Web.Enabled = true, want false")
	}
	if got, want := c.Web.ListenAddr, "127.0.0.1:19999"; got != want {
		t.Errorf("Web.ListenAddr = %q, want %q", got, want)
	}
}

func TestApplyEnv_Web(t *testing.T) {
	env := map[string]string{
		"REMOTELAUNCHER_WEB_ENABLED":     "false",
		"REMOTELAUNCHER_WEB_LISTEN_ADDR": "localhost:12345",
	}
	c := Defaults()
	if err := c.applyEnvFrom(func(k string) (string, bool) {
		v, ok := env[k]
		return v, ok
	}); err != nil {
		t.Fatalf("applyEnvFrom: %v", err)
	}
	if c.Web.Enabled {
		t.Error("Web.Enabled = true, want false")
	}
	if got, want := c.Web.ListenAddr, "localhost:12345"; got != want {
		t.Errorf("Web.ListenAddr = %q, want %q", got, want)
	}
}

func TestApplyEnv_WebBadBool(t *testing.T) {
	env := map[string]string{"REMOTELAUNCHER_WEB_ENABLED": "not-a-bool"}
	c := Defaults()
	err := c.applyEnvFrom(func(k string) (string, bool) {
		v, ok := env[k]
		return v, ok
	})
	if err == nil {
		t.Fatal("applyEnvFrom: want error, got nil")
	}
}

func TestApplyFlags_Web(t *testing.T) {
	c := Defaults()
	if err := c.ApplyFlags([]string{
		"-web-enabled=false",
		"-web-listen=127.0.0.1:11111",
	}); err != nil {
		t.Fatalf("ApplyFlags: %v", err)
	}
	if c.Web.Enabled {
		t.Error("Web.Enabled = true, want false")
	}
	if got, want := c.Web.ListenAddr, "127.0.0.1:11111"; got != want {
		t.Errorf("Web.ListenAddr = %q, want %q", got, want)
	}
}
