// SPDX-FileCopyrightText: William Moreno Reyes CP | MBA
// SPDX-License-Identifier: Apache-2.0

package output

import (
	"bytes"
	"strings"
	"testing"
)

func TestPrintJSONFormatsStructuredPayload(t *testing.T) {
	var out bytes.Buffer
	PrintJSON(&out, map[string]any{"status": "ok", "count": 2})

	if !strings.Contains(out.String(), "\"status\": \"ok\"") {
		t.Fatalf("expected formatted JSON output, got %q", out.String())
	}
	if !strings.Contains(out.String(), "\"count\": 2") {
		t.Fatalf("expected numeric field in output, got %q", out.String())
	}
}

func TestPrintJSONReportsMarshalError(t *testing.T) {
	var out bytes.Buffer
	PrintJSON(&out, make(chan int))

	if !strings.Contains(out.String(), "Error formatting JSON:") {
		t.Fatalf("expected marshal error output, got %q", out.String())
	}
}

func TestPrintTableRendersHeadersAndRows(t *testing.T) {
	var out bytes.Buffer
	PrintTable(&out, []string{"NAME", "STATUS"}, [][]string{
		{"node-1", "healthy"},
		{"node-2", "degraded"},
	})

	for _, want := range []string{"NAME", "STATUS", "node-1", "healthy", "node-2", "degraded"} {
		if !strings.Contains(out.String(), want) {
			t.Fatalf("expected %q in table output, got %q", want, out.String())
		}
	}
}
