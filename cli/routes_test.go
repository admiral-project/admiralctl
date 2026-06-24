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

func TestRoutesList(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`[{"hostname":"host1.example.com","route_kind":"http","app_instance_id":"inst1","service_name":"web","target_url":"http://10.0.0.1","status":"active"}]`))
	}))
	defer ts.Close()

	SetClient(client.NewWithHTTP(ts.URL, "token", ts.Client()))

	buf := new(bytes.Buffer)
	output.SetOut(buf)

	if err := runRoutesList(routesListCmd, nil); err != nil {
		t.Fatalf("runRoutesList: %v", err)
	}

	if !bytes.Contains(buf.Bytes(), []byte("host1.example.com")) {
		t.Errorf("expected output to contain host1.example.com, got %q", buf.String())
	}
}
