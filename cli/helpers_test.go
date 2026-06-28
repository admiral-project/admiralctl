// SPDX-FileCopyrightText: William Moreno Reyes CP | MBA
// SPDX-License-Identifier: Apache-2.0

package cli

import (
	"strings"
	"testing"
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
