// SPDX-FileCopyrightText: William Moreno Reyes CP | MBA
// SPDX-License-Identifier: Apache-2.0

package output

import (
	"bytes"
	"strings"
	"testing"
)

func TestPrintJSONFormatsStructuredPayload(t *testing.T) {
	buf := new(bytes.Buffer)
	SetOut(buf)

	PrintJSON(map[string]any{
		"status": "ok",
		"count":  2,
	})

	out := buf.String()
	if !strings.Contains(out, "\"status\": \"ok\"") {
		t.Fatalf("expected formatted JSON output, got %q", out)
	}
	if !strings.Contains(out, "\"count\": 2") {
		t.Fatalf("expected numeric field in output, got %q", out)
	}
}

func TestPrintJSONReportsMarshalError(t *testing.T) {
	buf := new(bytes.Buffer)
	SetOut(buf)

	PrintJSON(make(chan int))

	out := buf.String()
	if !strings.Contains(out, "Error formatting JSON:") {
		t.Fatalf("expected marshal error output, got %q", out)
	}
}

func TestPrintTableRendersHeadersAndRows(t *testing.T) {
	buf := new(bytes.Buffer)
	SetOut(buf)

	PrintTable(
		[]string{"NAME", "STATUS"},
		[][]string{
			{"node-1", "healthy"},
			{"node-2", "degraded"},
		},
	)

	out := buf.String()
	for _, want := range []string{"NAME", "STATUS", "node-1", "healthy", "node-2", "degraded"} {
		if !strings.Contains(out, want) {
			t.Fatalf("expected %q in table output, got %q", want, out)
		}
	}
}
