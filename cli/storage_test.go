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

func TestStorageInstancesCmd(t *testing.T) {
	httpClient := &http.Client{
		Transport: mockRoundTripper(func(r *http.Request) (*http.Response, error) {
			apps := []map[string]interface{}{
				{
					"id":                   "inst-1",
					"app_definition_name":  "erpnext",
					"storage_state":        "warning",
					"storage_exceeded":     false,
					"storage_used_bytes":   1024,
					"storage_limit_bytes":  4096,
					"storage_used_percent": 25.0,
					"grace_period_ends_at": "2023-01-01",
				},
			}
			body, _ := json.Marshal(apps)
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(bytes.NewReader(body)),
				Header:     make(http.Header),
			}, nil
		}),
	}

	c := newMockClient(t, httpClient)
	SetClient(c)

	var out bytes.Buffer
	storageInstancesCmd.SetOut(&out)

	err := runStorageInstances(storageInstancesCmd, nil)
	if err != nil {
		t.Fatalf("runStorageInstances failed: %v", err)
	}

	got := out.String()
	if !strings.Contains(got, "inst-1") || !strings.Contains(got, "erpnext") {
		t.Fatalf("unexpected output: %q", got)
	}
}

func TestStorageNodesCmd(t *testing.T) {
	httpClient := &http.Client{
		Transport: mockRoundTripper(func(r *http.Request) (*http.Response, error) {
			nodes := []map[string]interface{}{
				{
					"id":                   "node-1",
					"hostname":             "host-1",
					"disk_used_bytes":      100,
					"disk_total_bytes":     1000,
					"ram_used_bytes":       200,
					"ram_total_bytes":      2000,
					"committed_ram_bytes":  500,
					"committed_disk_bytes": 500,
					"health_status":        "healthy",
					"storage_state":        "warning",
				},
			}
			body, _ := json.Marshal(nodes)
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(bytes.NewReader(body)),
				Header:     make(http.Header),
			}, nil
		}),
	}

	c := newMockClient(t, httpClient)
	SetClient(c)

	var out bytes.Buffer
	storageNodesCmd.SetOut(&out)

	err := runStorageNodes(storageNodesCmd, nil)
	if err != nil {
		t.Fatalf("runStorageNodes failed: %v", err)
	}

	got := out.String()
	if !strings.Contains(got, "node-1") || !strings.Contains(got, "host-1") {
		t.Fatalf("unexpected output: %q", got)
	}
}
