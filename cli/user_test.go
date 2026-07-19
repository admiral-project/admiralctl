// SPDX-FileCopyrightText: William Moreno Reyes CP | MBA
// SPDX-License-Identifier: Apache-2.0

package cli

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"os"
	"strings"
	"testing"
)

func withMockStdin(t *testing.T, content string, fn func()) {
	t.Helper()
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("os.Pipe: %v", err)
	}
	defer r.Close()

	originalStdin := os.Stdin
	os.Stdin = r
	defer func() {
		os.Stdin = originalStdin
	}()

	_, err = w.Write([]byte(content))
	if err != nil {
		t.Fatalf("write stdin: %v", err)
	}
	w.Close()

	fn()
}

func TestUserCreateCmd(t *testing.T) {
	httpClient := &http.Client{
		Transport: mockRoundTripper(func(r *http.Request) (*http.Response, error) {
			if r.Method == "POST" && r.URL.Path == "/api/admin/users" {
				user := map[string]interface{}{
					"username": "newuser",
					"role":     "admin",
				}
				body, _ := json.Marshal(user)
				return &http.Response{
					StatusCode: http.StatusCreated,
					Body:       io.NopCloser(bytes.NewReader(body)),
					Header:     make(http.Header),
				}, nil
			}
			return &http.Response{
				StatusCode: http.StatusNotFound,
				Body:       io.NopCloser(strings.NewReader("")),
				Header:     make(http.Header),
			}, nil
		}),
	}

	c := newMockClient(t, httpClient)
	SetClient(c)

	var out bytes.Buffer
	userCreateCmd.SetOut(&out)
	_ = userCreateCmd.Flags().Set("type", "admin")

	withMockStdin(t, "mypassword\n", func() {
		err := runUserCreate(userCreateCmd, []string{"newuser"})
		if err != nil {
			t.Fatalf("runUserCreate failed: %v", err)
		}
	})

	got := out.String()
	if !strings.Contains(got, "created successfully") || !strings.Contains(got, "newuser") {
		t.Fatalf("unexpected output: %q", got)
	}
}

func TestUserSetPasswordCmd(t *testing.T) {
	httpClient := &http.Client{
		Transport: mockRoundTripper(func(r *http.Request) (*http.Response, error) {
			if r.Method == "POST" && r.URL.Path == "/api/admin/users/existinguser/set-password" {
				return &http.Response{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader("")),
					Header:     make(http.Header),
				}, nil
			}
			return &http.Response{
				StatusCode: http.StatusNotFound,
				Body:       io.NopCloser(strings.NewReader("")),
				Header:     make(http.Header),
			}, nil
		}),
	}

	c := newMockClient(t, httpClient)
	SetClient(c)

	var out bytes.Buffer
	userSetPasswordCmd.SetOut(&out)

	withMockStdin(t, "newpassword\n", func() {
		err := runUserSetPassword(userSetPasswordCmd, []string{"existinguser"})
		if err != nil {
			t.Fatalf("runUserSetPassword failed: %v", err)
		}
	})

	got := out.String()
	if !strings.Contains(got, "Password for user \"existinguser\" updated successfully") {
		t.Fatalf("unexpected output: %q", got)
	}
}
