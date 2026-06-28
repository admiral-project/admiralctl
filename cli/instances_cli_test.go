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

	"github.com/admiral-project/admiral/admiralctl/internal/client"
)

func TestInstancesListCmd(t *testing.T) {
	httpClient := &http.Client{
		Transport: mockRoundTripper(func(r *http.Request) (*http.Response, error) {
			instances := []map[string]interface{}{
				{
					"id":                   "inst-1",
					"customer_id":          "cust-1",
					"app_definition_name":  "erpnext",
					"tier_name":            "small",
					"node_id":              "node-1",
					"commercial_status":    "active",
					"technical_status":     "running",
					"storage_state":        "healthy",
				},
			}
			body, _ := json.Marshal(instances)
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(bytes.NewReader(body)),
				Header:     make(http.Header),
			}, nil
		}),
	}

	c := client.NewWithHTTP("https://localhost", "fake-token", httpClient)
	SetClient(c)

	got := captureStdout(func() {
		err := runInstancesList(instancesListCmd, nil)
		if err != nil {
			t.Fatalf("runInstancesList failed: %v", err)
		}
	})

	if !strings.Contains(got, "inst-1") || !strings.Contains(got, "cust-1") {
		t.Fatalf("expected inst-1 and cust-1 in output, got %q", got)
	}
}

func TestInstancesShowCmd(t *testing.T) {
	httpClient := &http.Client{
		Transport: mockRoundTripper(func(r *http.Request) (*http.Response, error) {
			instance := map[string]interface{}{
				"id": "inst-1",
			}
			body, _ := json.Marshal(instance)
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(bytes.NewReader(body)),
				Header:     make(http.Header),
			}, nil
		}),
	}

	c := client.NewWithHTTP("https://localhost", "fake-token", httpClient)
	SetClient(c)

	got := captureStdout(func() {
		err := runInstancesShow(instancesShowCmd, []string{"inst-1"})
		if err != nil {
			t.Fatalf("runInstancesShow failed: %v", err)
		}
	})

	if !strings.Contains(got, "\"id\": \"inst-1\"") {
		t.Fatalf("unexpected output: %q", got)
	}
}
