// SPDX-FileCopyrightText: William Moreno Reyes CP | MBA
// SPDX-License-Identifier: Apache-2.0

package cli

import (
	"testing"

	"github.com/spf13/cobra"
)

func TestSkipClientLoad(t *testing.T) {
	tests := []struct {
		cmdName string
		want    bool
	}{
		{"init", true},
		{"version", true},
		{"status", false},
		{"nodes", false},
	}

	for _, tt := range tests {
		got := skipClientLoad(&cobra.Command{Use: tt.cmdName})
		if got != tt.want {
			t.Errorf("skipClientLoad(%q) = %v, want %v", tt.cmdName, got, tt.want)
		}
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
