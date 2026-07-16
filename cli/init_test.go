// SPDX-FileCopyrightText: William Moreno Reyes CP | MBA
// SPDX-License-Identifier: Apache-2.0

package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestWriteSigningKeyCreatesPrivateFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "signing-key.seed")
	seed := []byte("private signing seed")

	if err := writeSigningKey(path, seed); err != nil {
		t.Fatalf("writeSigningKey: %v", err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read signing key: %v", err)
	}
	if string(data) != "70726976617465207369676e696e672073656564\n" {
		t.Fatalf("unexpected signing key contents: %q", data)
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("stat signing key: %v", err)
	}
	if info.Mode().Perm() != 0600 {
		t.Fatalf("signing key mode = %o, want 600", info.Mode().Perm())
	}
	if err := writeSigningKey(path, seed); err == nil {
		t.Fatal("expected existing signing key to be preserved")
	} else if !strings.Contains(err.Error(), "file exists") {
		t.Fatalf("unexpected existing-file error: %v", err)
	}
}
