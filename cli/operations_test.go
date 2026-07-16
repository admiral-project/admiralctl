// SPDX-FileCopyrightText: William Moreno Reyes CP | MBA
// SPDX-License-Identifier: Apache-2.0

package cli

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"
)

func TestOperationsListCmd(t *testing.T) {
	httpClient := &http.Client{
		Transport: mockRoundTripper(func(r *http.Request) (*http.Response, error) {
			ops := []map[string]interface{}{
				{
					"id":          "op-1",
					"instance_id": "inst-1",
					"action":      "provision",
					"status":      "succeeded",
					"updated_at":  "2023-01-01",
				},
			}
			body, _ := json.Marshal(ops)
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(bytes.NewReader(body)),
				Header:     make(http.Header),
			}, nil
		}),
	}

	c := newMockClient(t, httpClient)
	SetClient(c)

	got := captureStdout(func() {
		err := runOperationsList(operationsListCmd, nil)
		if err != nil {
			t.Fatalf("runOperationsList failed: %v", err)
		}
	})

	if !strings.Contains(got, "op-1") || !strings.Contains(got, "inst-1") {
		t.Fatalf("expected op-1 and inst-1 in output, got %q", got)
	}
}
