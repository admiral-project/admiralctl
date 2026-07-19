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

func TestBackupsShowCmd(t *testing.T) {
	httpClient := &http.Client{
		Transport: mockRoundTripper(func(r *http.Request) (*http.Response, error) {
			backup := map[string]interface{}{"id": "bk-show"}
			body, _ := json.Marshal(backup)
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
	backupsShowCmd.SetOut(&out)

	err := runBackupsShow(backupsShowCmd, []string{"bk-show"})
	if err != nil {
		t.Fatalf("runBackupsShow failed: %v", err)
	}

	got := out.String()
	if !strings.Contains(got, "\"id\": \"bk-show\"") {
		t.Fatalf("unexpected output: %q", got)
	}
}

func TestBackupsRestoreCmd(t *testing.T) {
	httpClient := &http.Client{
		Transport: mockRoundTripper(func(r *http.Request) (*http.Response, error) {
			res := map[string]interface{}{"operation_id": "op-restore"}
			body, _ := json.Marshal(res)
			return &http.Response{
				StatusCode: http.StatusAccepted,
				Body:       io.NopCloser(bytes.NewReader(body)),
				Header:     make(http.Header),
			}, nil
		}),
	}

	c := newMockClient(t, httpClient)
	SetClient(c)

	var out bytes.Buffer
	backupsRestoreCmd.SetOut(&out)
	_ = backupsRestoreCmd.Flags().Set("backup-id", "bk-restore")
	_ = backupsRestoreCmd.Flags().Set("instance-id", "inst-1")
	_ = backupsRestoreCmd.Flags().Set("service", "db")

	err := runBackupsRestore(backupsRestoreCmd, nil)
	if err != nil {
		t.Fatalf("runBackupsRestore failed: %v", err)
	}

	got := out.String()
	if !strings.Contains(got, "Restore queued successfully") || !strings.Contains(got, "op-restore") {
		t.Fatalf("unexpected output: %q", got)
	}
}

func TestBackupsStorageSetCmd(t *testing.T) {
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
	backupsStorageSetCmd.SetOut(&out)
	_ = backupsStorageSetCmd.Flags().Set("backend", "s3")
	_ = backupsStorageSetCmd.Flags().Set("bucket", "new-bucket")

	err := runBackupsStorageSet(backupsStorageSetCmd, nil)
	if err != nil {
		t.Fatalf("runBackupsStorageSet failed: %v", err)
	}

	got := out.String()
	if !strings.Contains(got, "Backup storage configuration updated") {
		t.Fatalf("unexpected output: %q", got)
	}
}

func TestBackupsStorageTestCmd(t *testing.T) {
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
	backupsStorageTestCmd.SetOut(&out)

	err := runBackupsStorageTest(backupsStorageTestCmd, nil)
	if err != nil {
		t.Fatalf("runBackupsStorageTest failed: %v", err)
	}

	got := out.String()
	if !strings.Contains(got, "Backup storage test passed") {
		t.Fatalf("unexpected output: %q", got)
	}
}

func TestBackupsDeleteCmd(t *testing.T) {
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
	backupsDeleteCmd.SetOut(&out)

	withMockStdin(t, "y\n", func() {
		err := runBackupsDelete(backupsDeleteCmd, []string{"bk-del"})
		if err != nil {
			t.Fatalf("runBackupsDelete failed: %v", err)
		}
	})

	got := out.String()
	if !strings.Contains(got, "Backup bk-del deleted") {
		t.Fatalf("unexpected output: %q", got)
	}
}

func TestBackupsPruneCmd(t *testing.T) {
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
	backupsPruneCmd.SetOut(&out)

	withMockStdin(t, "y\n", func() {
		err := runBackupsPrune(backupsPruneCmd, nil)
		if err != nil {
			t.Fatalf("runBackupsPrune failed: %v", err)
		}
	})

	got := out.String()
	if !strings.Contains(got, "Backups pruned successfully") {
		t.Fatalf("unexpected output: %q", got)
	}
}
