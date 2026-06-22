// SPDX-FileCopyrightText: William Moreno Reyes CP | MBA
// SPDX-License-Identifier: Apache-2.0

package output

import (
	"bytes"
	"os"
	"strings"
	"testing"
)

func captureStdout(t *testing.T, fn func()) string {
	t.Helper()

	original := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("create pipe: %v", err)
	}
	os.Stdout = w
	defer func() {
		os.Stdout = original
	}()

	fn()

	if err := w.Close(); err != nil {
		t.Fatalf("close writer: %v", err)
	}

	var buf bytes.Buffer
	if _, err := buf.ReadFrom(r); err != nil {
		t.Fatalf("read captured stdout: %v", err)
	}
	return buf.String()
}

func TestPrintJSONFormatsStructuredPayload(t *testing.T) {
	out := captureStdout(t, func() {
		PrintJSON(map[string]any{
			"status": "ok",
			"count":  2,
		})
	})

	if !strings.Contains(out, "\"status\": \"ok\"") {
		t.Fatalf("expected formatted JSON output, got %q", out)
	}
	if !strings.Contains(out, "\"count\": 2") {
		t.Fatalf("expected numeric field in output, got %q", out)
	}
}

func TestPrintJSONReportsMarshalError(t *testing.T) {
	out := captureStdout(t, func() {
		PrintJSON(make(chan int))
	})

	if !strings.Contains(out, "Error formatting JSON:") {
		t.Fatalf("expected marshal error output, got %q", out)
	}
}

func TestPrintTableRendersHeadersAndRows(t *testing.T) {
	out := captureStdout(t, func() {
		PrintTable(
			[]string{"NAME", "STATUS"},
			[][]string{
				{"node-1", "healthy"},
				{"node-2", "degraded"},
			},
		)
	})

	for _, want := range []string{"NAME", "STATUS", "node-1", "healthy", "node-2", "degraded"} {
		if !strings.Contains(out, want) {
			t.Fatalf("expected %q in table output, got %q", want, out)
		}
	}
}
