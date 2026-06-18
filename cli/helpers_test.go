// SPDX-FileCopyrightText: William Moreno Reyes CP | MBA
// SPDX-License-Identifier: Apache-2.0

package cli

import (
	"strings"
	"testing"
)

func TestReadPasswordFromReaderTrimsNewlines(t *testing.T) {
	password, err := readPasswordFromReader(strings.NewReader("super-secret\r\n"))
	if err != nil {
		t.Fatalf("readPasswordFromReader: %v", err)
	}
	if password != "super-secret" {
		t.Fatalf("unexpected password %q", password)
	}
}
