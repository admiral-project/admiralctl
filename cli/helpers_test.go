// SPDX-FileCopyrightText: William Moreno Reyes CP | MBA
// SPDX-License-Identifier: Apache-2.0

package cli

import (
	"bytes"
	"strings"
	"testing"

	"github.com/admiral-project/admiral/admirald/pkg/admiral"
	"github.com/spf13/cobra"
)

func TestReadPasswordFromReaderTrimsNewlines(t *testing.T) {
	input := "secret\n"
	got, err := readPasswordFromReader(strings.NewReader(input))
	if err != nil {
		t.Fatalf("readPasswordFromReader: %v", err)
	}
	if got != "secret" {
		t.Fatalf("expected secret, got %q", got)
	}
}

func TestSanitizeInputFilePath(t *testing.T) {
	tests := []struct {
		input    string
		expected string
		wantErr  bool
	}{
		{"/tmp/test", "/tmp/test", false},
		{"test.yaml", "test.yaml", false},
		{"", "", true},
		{"/path/../to/file", "/to/file", false},
	}

	for _, tt := range tests {
		got, err := sanitizeInputFilePath(tt.input)
		if (err != nil) != tt.wantErr {
			t.Errorf("sanitizeInputFilePath(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
			continue
		}
		if err == nil && got != tt.expected {
			t.Errorf("sanitizeInputFilePath(%q) = %q, want %q", tt.input, got, tt.expected)
		}
	}
}

func TestConfirmDestructive(t *testing.T) {
	cmd := &cobra.Command{}
	cmd.Flags().Bool("force", false, "force")

	// test --force flag
	_ = cmd.Flags().Set("force", "true")
	if !confirmDestructive(cmd, "delete", "thing") {
		t.Fatal("expected confirmDestructive with force=true to return true")
	}

	// test interactive confirmation "y"
	_ = cmd.Flags().Set("force", "false")
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetIn(strings.NewReader("y\n"))
	if !confirmDestructive(cmd, "delete", "thing") {
		t.Fatal("expected confirmDestructive with 'y' input to return true")
	}

	// test interactive confirmation "n"
	cmd.SetIn(strings.NewReader("n\n"))
	if confirmDestructive(cmd, "delete", "thing") {
		t.Fatal("expected confirmDestructive with 'n' input to return false")
	}
}

func TestResolveToken(t *testing.T) {
	cmd := &cobra.Command{}
	var out, errOut bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&errOut)

	// Explicit token flag
	tok, err := resolveToken(cmd, "flag-token", "cfg-token")
	if err != nil || tok != "flag-token" {
		t.Fatalf("unexpected token resolved: %s, err: %v", tok, err)
	}

	// Configured token
	tok, err = resolveToken(cmd, "", "cfg-token")
	if err != nil || tok != "cfg-token" {
		t.Fatalf("unexpected token resolved: %s, err: %v", tok, err)
	}
}

func TestPrintPolicyRejected(t *testing.T) {
	cmd := &cobra.Command{}
	var out bytes.Buffer
	cmd.SetOut(&out)

	resp := admiral.ProvisioningRejectedResponse{
		Code:            "no_capacity",
		Message:         "No nodes available",
		OperationID:     "op-1",
		TaskID:          "task-1",
		RequestedNodeID: "node-1",
		NodeEvaluations: []admiral.NodeProvisioningEvaluation{
			{NodeID: "node-1", Eligible: false, RejectionReasons: []string{"reasons"}},
		},
	}

	printPolicyRejected(cmd, resp)
	got := out.String()
	if !strings.Contains(got, "blocked by policy") || !strings.Contains(got, "node-1") || !strings.Contains(got, "reasons") {
		t.Fatalf("unexpected output: %q", got)
	}
}
