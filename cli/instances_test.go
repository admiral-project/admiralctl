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

func TestInstancesList(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`[{"id":"inst1","customer_id":"cust1","app_definition_name":"app1","tier_name":"small","node_id":"node1","commercial_status":"active","technical_status":"running","storage_state":"ok"}]`))
	}))
	defer ts.Close()

	SetClient(client.NewWithHTTP(ts.URL, "token", ts.Client()))

	buf := new(bytes.Buffer)
	output.SetOut(buf)
	instancesListCmd.SetOut(buf)

	if err := runInstancesList(instancesListCmd, nil); err != nil {
		t.Fatalf("runInstancesList: %v", err)
	}

	if !bytes.Contains(buf.Bytes(), []byte("inst1")) {
		t.Errorf("expected output to contain inst1, got %q", buf.String())
	}
}
