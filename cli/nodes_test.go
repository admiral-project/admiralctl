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

	c := newMockClient(t, httpClient)
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

	c := newMockClient(t, httpClient)
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

func TestNodesShowCmd(t *testing.T) {
	httpClient := &http.Client{
		Transport: mockRoundTripper(func(r *http.Request) (*http.Response, error) {
			node := map[string]interface{}{"id": "node-show"}
			body, _ := json.Marshal(node)
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
	nodesShowCmd.SetOut(&out)

	err := runNodesShow(nodesShowCmd, []string{"node-show"})
	if err != nil {
		t.Fatalf("runNodesShow failed: %v", err)
	}

	got := out.String()
	if !strings.Contains(got, "\"id\": \"node-show\"") {
		t.Fatalf("unexpected output: %q", got)
	}
}

func TestNodesEnableCmd(t *testing.T) {
	httpClient := &http.Client{
		Transport: mockRoundTripper(func(r *http.Request) (*http.Response, error) {
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(strings.NewReader("")),
				Header:     make(http.Header),
			}, nil
		}),
	}

	c := newMockClient(t, httpClient)
	SetClient(c)

	var out bytes.Buffer
	nodesEnableCmd.SetOut(&out)

	err := runNodesEnable(nodesEnableCmd, []string{"node-enable"})
	if err != nil {
		t.Fatalf("runNodesEnable failed: %v", err)
	}

	got := out.String()
	if !strings.Contains(got, "Node \"node-enable\" enabled") {
		t.Fatalf("unexpected output: %q", got)
	}
}

func TestNodesDisableCmd(t *testing.T) {
	httpClient := &http.Client{
		Transport: mockRoundTripper(func(r *http.Request) (*http.Response, error) {
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(strings.NewReader("")),
				Header:     make(http.Header),
			}, nil
		}),
	}

	c := newMockClient(t, httpClient)
	SetClient(c)

	var out bytes.Buffer
	nodesDisableCmd.SetOut(&out)
	_ = nodesDisableCmd.Flags().Set("force", "true")

	err := runNodesDisable(nodesDisableCmd, []string{"node-disable"})
	if err != nil {
		t.Fatalf("runNodesDisable failed: %v", err)
	}

	got := out.String()
	if !strings.Contains(got, "Node \"node-disable\" disabled") {
		t.Fatalf("unexpected output: %q", got)
	}
}

func TestNodesRemoveCmd(t *testing.T) {
	httpClient := &http.Client{
		Transport: mockRoundTripper(func(r *http.Request) (*http.Response, error) {
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(strings.NewReader("")),
				Header:     make(http.Header),
			}, nil
		}),
	}

	c := newMockClient(t, httpClient)
	SetClient(c)

	var out bytes.Buffer
	nodesRemoveCmd.SetOut(&out)
	_ = nodesRemoveCmd.Flags().Set("force", "true")

	err := runNodesRemove(nodesRemoveCmd, []string{"node-remove"})
	if err != nil {
		t.Fatalf("runNodesRemove failed: %v", err)
	}

	got := out.String()
	if !strings.Contains(got, "Node \"node-remove\" removed") {
		t.Fatalf("unexpected output: %q", got)
	}
}

func TestNodesReadyCmd(t *testing.T) {
	httpClient := &http.Client{
		Transport: mockRoundTripper(func(r *http.Request) (*http.Response, error) {
			res := map[string]interface{}{"role": "worker", "ready": true}
			body, _ := json.Marshal(res)
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
	nodesReadyCmd.SetOut(&out)
	_ = nodesReadyCmd.Flags().Set("node", "node-ready")

	err := runNodesReady(nodesReadyCmd, nil)
	if err != nil {
		t.Fatalf("runNodesReady failed: %v", err)
	}

	got := out.String()
	if !strings.Contains(got, "ready") || !strings.Contains(got, "node-ready") {
		t.Fatalf("unexpected output: %q", got)
	}
}
