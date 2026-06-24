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

func TestUserList(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`[{"username":"user1","role":"admin","created_at":"2023-01-01"}]`))
	}))
	defer ts.Close()

	SetClient(client.NewWithHTTP(ts.URL, "token", ts.Client()))

	buf := new(bytes.Buffer)
	output.SetOut(buf)
	userListCmd.SetOut(buf)

	if err := runUserList(userListCmd, nil); err != nil {
		t.Fatalf("runUserList: %v", err)
	}

	if !bytes.Contains(buf.Bytes(), []byte("user1")) {
		t.Errorf("expected output to contain user1, got %q", buf.String())
	}
}
