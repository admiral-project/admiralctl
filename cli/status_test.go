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

func TestStatusCmd(t *testing.T) {
	httpClient := &http.Client{
		Transport: mockRoundTripper(func(r *http.Request) (*http.Response, error) {
			status := map[string]interface{}{"status": "healthy"}
			body, _ := json.Marshal(status)
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
	statusCmd.SetOut(&out)

	err := runStatus(statusCmd, nil)
	if err != nil {
		t.Fatalf("runStatus failed: %v", err)
	}

	got := out.String()
	if !strings.Contains(got, "Control Plane:    healthy") {
		t.Fatalf("unexpected output: %q", got)
	}
}

func TestUserListCmd(t *testing.T) {
	httpClient := &http.Client{
		Transport: mockRoundTripper(func(r *http.Request) (*http.Response, error) {
			users := []map[string]interface{}{
				{"username": "admin", "role": "admin", "created_at": "2023-01-01"},
			}
			body, _ := json.Marshal(users)
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
		err := runUserList(userListCmd, nil)
		if err != nil {
			t.Fatalf("runUserList failed: %v", err)
		}
	})

	if !strings.Contains(got, "admin") {
		t.Fatalf("expected admin in output, got %q", got)
	}
}
