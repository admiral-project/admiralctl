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

func TestNodesListCmd(t *testing.T) {
	httpClient := &http.Client{
		Transport: mockRoundTripper(func(r *http.Request) (*http.Response, error) {
			nodes := []map[string]interface{}{
				{
					"id":                         "node-1",
					"hostname":                   "host-1",
					"status":                     "online",
					"health_status":              "healthy",
					"available_for_provisioning": true,
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

	c := client.NewWithHTTP("https://localhost", "fake-token", httpClient)
	SetClient(c)

	got := captureStdout(func() {
		err := runNodesList(nodesListCmd, nil)
		if err != nil {
			t.Fatalf("runNodesList failed: %v", err)
		}
	})

	if !strings.Contains(got, "node-1") || !strings.Contains(got, "host-1") {
		t.Fatalf("expected node-1 and host-1 in output, got %q", got)
	}
}

func TestNodesRegisterCmd(t *testing.T) {
	t.Setenv("ADMIRAL_NODE_TOKEN", "node-env-token")
	if err := nodesRegisterCmd.Flags().Set("token", ""); err != nil {
		t.Fatalf("reset token flag: %v", err)
	}
	httpClient := &http.Client{
		Transport: mockRoundTripper(func(r *http.Request) (*http.Response, error) {
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(strings.NewReader("")),
				Header:     make(http.Header),
			}, nil
		}),
	}

	c := client.NewWithHTTP("https://localhost", "fake-token", httpClient)
	SetClient(c)

	var out bytes.Buffer
	nodesRegisterCmd.SetOut(&out)
	nodesRegisterCmd.Flags().Set("id", "new-node")
	nodesRegisterCmd.Flags().Set("hostname", "new-host")
	nodesRegisterCmd.Flags().Set("ip", "1.1.1.1")

	err := runNodesRegister(nodesRegisterCmd, nil)
	if err != nil {
		t.Fatalf("runNodesRegister failed: %v", err)
	}

	got := out.String()
	if !strings.Contains(got, "registered successfully") {
		t.Fatalf("unexpected output: %q", got)
	}
}
