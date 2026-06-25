// SPDX-FileCopyrightText: William Moreno Reyes CP | MBA
// SPDX-License-Identifier: Apache-2.0

package cli

import (
	"path/filepath"
	"runtime"
	"testing"
)

func TestReadAndValidateERPNextExampleAppFile(t *testing.T) {
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	examplePath := filepath.Join(filepath.Dir(file), "..", "..", "examples", "apps", "erpnext.yaml")

	_, payload, err := readAndValidateAppFile(nil, examplePath)
	if err != nil {
		t.Fatalf("readAndValidateAppFile: %v", err)
	}
	if payload.Name != "erpnext" {
		t.Fatalf("expected erpnext payload, got %q", payload.Name)
	}
}
