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
