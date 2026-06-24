// SPDX-FileCopyrightText: William Moreno Reyes CP | MBA
// SPDX-License-Identifier: Apache-2.0

package config

import (
	"os"
	"testing"
)

func TestLoadWithoutConfigDoesNotInjectDefaultToken(t *testing.T) {
	setEnv(t, "HOME", t.TempDir())
	setEnv(t, "ADMIRAL_SERVER_URL", "")
	setEnv(t, "ADMIRAL_ADMIN_TOKEN", "")
	setEnv(t, "ADMIRAL_TLS_CA_FILE", "")

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

func TestSave(t *testing.T) {
	tmpDir := t.TempDir()
	setEnv(t, "HOME", tmpDir)

	cfg := &Config{
		ServerURL: "https://admiral.test",
		Token:     "test-token",
	}

	if err := Save(cfg); err != nil {
		t.Fatalf("Save: %v", err)
	}

	loaded, err := Load()
	if err != nil {
		t.Fatalf("Load after Save: %v", err)
	}

	if loaded.ServerURL != cfg.ServerURL || loaded.Token != cfg.Token {
		t.Errorf("loaded config does not match saved config")
	}
}
