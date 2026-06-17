// SPDX-FileCopyrightText: William Moreno Reyes CP | MBA
// SPDX-License-Identifier: Apache-2.0

package config

import (
	"os"
	"path/filepath"

	"github.com/admiral-project/admiral/admirald/pkg/admiral/tlsconfig"
	"gopkg.in/yaml.v2"
)

type Config struct {
	ServerURL  string `yaml:"server_url"`
	Token      string `yaml:"token"`
	CACertFile string `yaml:"ca_cert_file,omitempty"`
	Operator   string `yaml:"operator,omitempty"`
}

func GetConfigPath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		home = "."
	}
	return filepath.Join(home, ".config", "admiralctl", "config.yaml")
}

func Load() (*Config, error) {
	cfg := &Config{
		ServerURL: "https://localhost:8080",
	}

	path := GetConfigPath()
	data, err := os.ReadFile(path) // #nosec G304 -- config path is fixed by GetConfigPath()
	if err != nil {
		if !os.IsNotExist(err) {
			return nil, err
		}
	} else {
		if err := yaml.Unmarshal(data, cfg); err != nil {
			return nil, err
		}
	}

	if val := os.Getenv("ADMIRAL_SERVER_URL"); val != "" {
		cfg.ServerURL = val
	}
	if val := os.Getenv("ADMIRAL_ADMIN_TOKEN"); val != "" {
		cfg.Token = val
	}
	if val := os.Getenv("ADMIRAL_TLS_CA_FILE"); val != "" {
		cfg.CACertFile = val
	}
	if val := os.Getenv("ADMIRAL_OPERATOR"); val != "" {
		cfg.Operator = val
	}
	if err := tlsconfig.ValidateURLScheme(cfg.ServerURL, "https"); err != nil {
		return nil, err
	}

	return cfg, nil
}

func Save(cfg *Config) error {
	path := GetConfigPath()
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0750); err != nil {
		return err
	}

	data, err := yaml.Marshal(cfg)
	if err != nil {
		return err
	}

	return os.WriteFile(path, data, 0600)
}
