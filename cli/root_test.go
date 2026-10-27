// SPDX-FileCopyrightText: William Moreno Reyes CP | MBA
// SPDX-License-Identifier: Apache-2.0

package cli

import (
	"testing"

	"github.com/spf13/cobra"
)

func TestSkipClientLoad(t *testing.T) {
	tests := []struct {
		name string
		cmd  *cobra.Command
		want bool
	}{
		{"init", &cobra.Command{Use: "init"}, true},
		{"version", &cobra.Command{Use: "version"}, true},
		{"apps validate", appsValidateCmd, true},
		{"apps apply", appsApplyCmd, false},
		{"status", &cobra.Command{Use: "status"}, false},
		{"nodes", &cobra.Command{Use: "nodes"}, false},
	}

	for _, tt := range tests {
		got := skipClientLoad(tt.cmd)
		if got != tt.want {
			t.Errorf("skipClientLoad(%q) = %v, want %v", tt.name, got, tt.want)
		}
	}
}

func TestLoadClientSkipsAppValidationWithoutConfiguration(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	t.Setenv("ADMIRAL_OPERATOR_TOKEN", "")
	t.Setenv("ADMIRAL_INTERNAL_TOKEN", "")

	previousConfig := cfg
	previousClient := currentClient
	cfg = nil
	currentClient = nil
	t.Cleanup(func() {
		cfg = previousConfig
		currentClient = previousClient
	})

	if err := loadClient(appsValidateCmd, nil); err != nil {
		t.Fatalf("loadClient(apps validate) returned error without CLI configuration: %v", err)
	}
	if cfg != nil || currentClient != nil {
		t.Fatal("apps validate should not load CLI configuration or create an API client")
	}
}

func TestValidateOutputFlag(t *testing.T) {
	cmd := &cobra.Command{}
	cmd.Flags().String("output", "table", "output format")

	for _, value := range []string{"table", "json"} {
		if err := cmd.Flags().Set("output", value); err != nil {
			t.Fatalf("set output: %v", err)
		}
		if err := validateOutputFlag(cmd); err != nil {
			t.Errorf("validateOutputFlag(%q) returned error: %v", value, err)
		}
	}

	if err := cmd.Flags().Set("output", "yaml"); err != nil {
		t.Fatalf("set invalid output: %v", err)
	}
	if err := validateOutputFlag(cmd); err == nil {
		t.Fatal("validateOutputFlag accepted invalid output")
	}
}
