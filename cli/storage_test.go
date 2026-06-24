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

func TestStorageInstances(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`[{"id":"inst1","app_definition_name":"app1","storage_state":"ok"}]`))
	}))
	defer ts.Close()

	SetClient(client.NewWithHTTP(ts.URL, "token", ts.Client()))

	buf := new(bytes.Buffer)
	output.SetOut(buf)
	storageInstancesCmd.SetOut(buf)

	if err := runStorageInstances(storageInstancesCmd, nil); err != nil {
		t.Fatalf("runStorageInstances: %v", err)
	}

	if !bytes.Contains(buf.Bytes(), []byte("inst1")) {
		t.Errorf("expected output to contain inst1, got %q", buf.String())
	}
}

func TestStorageNodes(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`[{"id":"node1","hostname":"host1"}]`))
	}))
	defer ts.Close()

	SetClient(client.NewWithHTTP(ts.URL, "token", ts.Client()))

	buf := new(bytes.Buffer)
	output.SetOut(buf)
	storageNodesCmd.SetOut(buf)

	if err := runStorageNodes(storageNodesCmd, nil); err != nil {
		t.Fatalf("runStorageNodes: %v", err)
	}

	if !bytes.Contains(buf.Bytes(), []byte("node1")) {
		t.Errorf("expected output to contain node1, got %q", buf.String())
	}
}
