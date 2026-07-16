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

func TestAppsListCmd(t *testing.T) {
	httpClient := &http.Client{
		Transport: mockRoundTripper(func(r *http.Request) (*http.Response, error) {
			apps := []map[string]interface{}{
				{
					"name":         "erpnext",
					"display_name": "ERPNext",
					"status":       "active",
					"created_at":   "2023-01-01",
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

	got := captureStdout(func() {
		err := runAppsList(appsListCmd, nil)
		if err != nil {
			t.Fatalf("runAppsList failed: %v", err)
		}
	})

	if !strings.Contains(got, "erpnext") || !strings.Contains(got, "ERPNext") {
		t.Fatalf("expected erpnext and ERPNext in output, got %q", got)
	}
}

func TestAppsShowCmd(t *testing.T) {
	httpClient := &http.Client{
		Transport: mockRoundTripper(func(r *http.Request) (*http.Response, error) {
			app := map[string]interface{}{
				"name": "erpnext",
			}
			body, _ := json.Marshal(app)
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
		err := runAppsShow(appsShowCmd, []string{"erpnext"})
		if err != nil {
			t.Fatalf("runAppsShow failed: %v", err)
		}
	})

	if !strings.Contains(got, "\"name\": \"erpnext\"") {
		t.Fatalf("unexpected output: %q", got)
	}
}
