// SPDX-FileCopyrightText: William Moreno Reyes CP | MBA
// SPDX-License-Identifier: Apache-2.0

package cli

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/admiral-project/admiral/admiralctl/internal/client"
	"github.com/admiral-project/admiral/admiralctl/internal/output"
)

func TestBackupsList(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`[{"id":"bk1","instance_id":"inst1","backup_type":"full","storage_backend":"s3","status":"succeeded","created_at":"2023-01-01"}]`))
	}))
	defer ts.Close()

	SetClient(client.NewWithHTTP(ts.URL, "token", ts.Client()))

	buf := new(bytes.Buffer)
	output.SetOut(buf)
	backupsListCmd.SetOut(buf)

	if err := runBackupsList(backupsListCmd, nil); err != nil {
		t.Fatalf("runBackupsList: %v", err)
	}

	if !bytes.Contains(buf.Bytes(), []byte("bk1")) {
		t.Errorf("expected output to contain bk1, got %q", buf.String())
	}
}
