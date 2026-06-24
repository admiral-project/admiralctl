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

func TestNodesList(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`[{"id":"node1","hostname":"host1","status":"online"}]`))
	}))
	defer ts.Close()

	SetClient(client.NewWithHTTP(ts.URL, "token", ts.Client()))

	buf := new(bytes.Buffer)
	output.SetOut(buf)
	nodesListCmd.SetOut(buf)

	if err := runNodesList(nodesListCmd, nil); err != nil {
		t.Fatalf("runNodesList: %v", err)
	}

	if !bytes.Contains(buf.Bytes(), []byte("node1")) {
		t.Errorf("expected output to contain node1, got %q", buf.String())
	}
}

func TestNodesShow(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"id":"node1"}`))
	}))
	defer ts.Close()

	SetClient(client.NewWithHTTP(ts.URL, "token", ts.Client()))

	buf := new(bytes.Buffer)
	output.SetOut(buf)
	nodesShowCmd.SetOut(buf)

	if err := runNodesShow(nodesShowCmd, []string{"node1"}); err != nil {
		t.Fatalf("runNodesShow: %v", err)
	}

	if !bytes.Contains(buf.Bytes(), []byte("node1")) {
		t.Errorf("expected output to contain node1, got %q", buf.String())
	}
}
