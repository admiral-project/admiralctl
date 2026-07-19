// SPDX-FileCopyrightText: William Moreno Reyes CP | MBA
// SPDX-License-Identifier: Apache-2.0

package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoadWithoutConfigDoesNotInjectDefaultToken(t *testing.T) {
	setEnv(t, "HOME", t.TempDir())
	setEnv(t, "ADMIRAL_SERVER_URL", "")
	setEnv(t, "ADMIRAL_ADMIN_TOKEN", "")
	setEnv(t, "ADMIRAL_TLS_CA_FILE", "")
	setEnv(t, "ADMIRAL_OPERATOR", "")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("load returned error: %v", err)
	}
	if cfg.ServerURL != "https://localhost:8080" {
		t.Fatalf("expected default server URL, got %q", cfg.ServerURL)
	}
	if cfg.Token != "" {
		t.Fatalf("expected empty token when not configured, got %q", cfg.Token)
	}
}

func TestLoadAppliesEnvironmentOverrides(t *testing.T) {
	setEnv(t, "HOME", t.TempDir())
	setEnv(t, "ADMIRAL_SERVER_URL", "https://admiral.example.com")
	setEnv(t, "ADMIRAL_ADMIN_TOKEN", "env-token")
	setEnv(t, "ADMIRAL_TLS_CA_FILE", "/etc/ssl/admiral-ca.pem")
	setEnv(t, "ADMIRAL_OPERATOR", "jules")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("load returned error: %v", err)
	}
	if cfg.ServerURL != "https://admiral.example.com" {
		t.Fatalf("expected server URL override, got %q", cfg.ServerURL)
	}
	if cfg.Token != "env-token" {
		t.Fatalf("expected token override, got %q", cfg.Token)
	}
	if cfg.CACertFile != "/etc/ssl/admiral-ca.pem" {
		t.Fatalf("expected CA override, got %q", cfg.CACertFile)
	}
	if cfg.Operator != "jules" {
		t.Fatalf("expected operator override, got %q", cfg.Operator)
	}
}

func TestLoadRejectsHTTPServerURL(t *testing.T) {
	setEnv(t, "HOME", t.TempDir())
	setEnv(t, "ADMIRAL_SERVER_URL", "http://localhost:8080")
	setEnv(t, "ADMIRAL_ADMIN_TOKEN", "")
	setEnv(t, "ADMIRAL_TLS_CA_FILE", "")

	_, err := Load()
	if err == nil {
		t.Fatal("expected error for plaintext server URL")
	}
}

func TestSaveAndLoadConfig(t *testing.T) {
	tempHome := t.TempDir()
	setEnv(t, "HOME", tempHome)
	setEnv(t, "ADMIRAL_SERVER_URL", "")
	setEnv(t, "ADMIRAL_ADMIN_TOKEN", "")
	setEnv(t, "ADMIRAL_TLS_CA_FILE", "")
	setEnv(t, "ADMIRAL_OPERATOR", "")

	expected := &Config{
		ServerURL:  "https://admiral.test",
		Token:      "test-token",
		CACertFile: "/tmp/ca.pem",
		Operator:   "test-user",
	}

	if err := Save(expected); err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	if cfg.ServerURL != expected.ServerURL || cfg.Token != expected.Token ||
		cfg.CACertFile != expected.CACertFile || cfg.Operator != expected.Operator {
		t.Fatalf("Loaded config does not match saved config: %+v", cfg)
	}
}

func TestLoadInvalidYAML(t *testing.T) {
	tempHome := t.TempDir()
	setEnv(t, "HOME", tempHome)
	setEnv(t, "ADMIRAL_SERVER_URL", "")
	setEnv(t, "ADMIRAL_ADMIN_TOKEN", "")

	configPath := GetConfigPath()
	if err := os.MkdirAll(filepath.Dir(configPath), 0750); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(configPath, []byte("invalid: yaml: :"), 0600); err != nil {
		t.Fatal(err)
	}

	_, err := Load()
	if err == nil {
		t.Fatal("expected error when loading invalid YAML")
	}
}

func TestGetSigningKeyPath(t *testing.T) {
	tempHome := t.TempDir()
	setEnv(t, "HOME", tempHome)
	path := GetSigningKeyPath()
	if !strings.HasSuffix(path, filepath.Join("admiralctl", "signing-key.seed")) {
		t.Fatalf("GetSigningKeyPath returned unexpected path: %s", path)
	}
}

func TestGetConfigPathHomeError(t *testing.T) {
	setEnv(t, "HOME", "")
	// Unset other potential home-related variables on different OSes if necessary
	setEnv(t, "USERPROFILE", "")

	path := GetConfigPath()
	// Should fallback to current directory
	if !strings.HasSuffix(path, filepath.Join(".config", "admiralctl", "config.yaml")) {
		t.Fatalf("GetConfigPath with empty Home returned unexpected path: %s", path)
	}
}

func setEnv(t *testing.T, key, value string) {
	t.Helper()

	original, ok := os.LookupEnv(key)
	if err := os.Setenv(key, value); err != nil {
		t.Fatalf("set %s: %v", key, err)
	}

	t.Cleanup(func() {
		var err error
		if ok {
			err = os.Setenv(key, original)
		} else {
			err = os.Unsetenv(key)
		}
		if err != nil {
			t.Fatalf("restore %s: %v", key, err)
		}
	})
}
