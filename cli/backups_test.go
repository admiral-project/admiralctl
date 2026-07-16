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

func TestBackupsListCmd(t *testing.T) {
	httpClient := &http.Client{
		Transport: mockRoundTripper(func(r *http.Request) (*http.Response, error) {
			backups := []map[string]interface{}{
				{
					"id":              "bk-1",
					"instance_id":     "inst-1",
					"backup_type":     "manual",
					"storage_backend": "s3",
					"status":          "succeeded",
					"created_at":      "2023-01-01",
				},
			}
			body, _ := json.Marshal(backups)
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
		err := runBackupsList(backupsListCmd, nil)
		if err != nil {
			t.Fatalf("runBackupsList failed: %v", err)
		}
	})

	if !strings.Contains(got, "bk-1") || !strings.Contains(got, "inst-1") {
		t.Fatalf("expected bk-1 and inst-1 in output, got %q", got)
	}
}

func TestBackupsStorageGetCmd(t *testing.T) {
	httpClient := &http.Client{
		Transport: mockRoundTripper(func(r *http.Request) (*http.Response, error) {
			cfg := map[string]interface{}{
				"backend": "s3",
				"enabled": true,
				"bucket":  "my-bucket",
			}
			body, _ := json.Marshal(cfg)
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
		err := runBackupsStorageGet(backupsStorageGetCmd, nil)
		if err != nil {
			t.Fatalf("runBackupsStorageGet failed: %v", err)
		}
	})

	if !strings.Contains(got, "Backend:  s3") || !strings.Contains(got, "Bucket:   my-bucket") {
		t.Fatalf("unexpected output: %q", got)
	}
}
