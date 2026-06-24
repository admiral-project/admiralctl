// SPDX-FileCopyrightText: William Moreno Reyes CP | MBA
// SPDX-License-Identifier: Apache-2.0

package cli

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/admiral-project/admiral/admiralctl/internal/client"
	"github.com/admiral-project/admiral/admiralctl/internal/output"
)

func TestMain(m *testing.M) {
	os.Setenv("ADMIRAL_ADMIN_TOKEN", "test-token")
	os.Exit(m.Run())
}

func TestAppsList(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`[{"name":"myapp","display_name":"My App","status":"active","created_at":"2023-01-01"}]`))
	}))
	defer ts.Close()

	SetClient(client.NewWithHTTP(ts.URL, "token", ts.Client()))

	buf := new(bytes.Buffer)
	output.SetOut(buf)

	if err := runAppsList(appsListCmd, nil); err != nil {
		t.Fatalf("runAppsList: %v", err)
	}

	if !bytes.Contains(buf.Bytes(), []byte("myapp")) {
		t.Errorf("expected output to contain myapp, got %q", buf.String())
	}
}

func TestAppsShow(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"name":"myapp"}`))
	}))
	defer ts.Close()

	SetClient(client.NewWithHTTP(ts.URL, "token", ts.Client()))

	buf := new(bytes.Buffer)
	output.SetOut(buf)

	if err := runAppsShow(appsShowCmd, []string{"myapp"}); err != nil {
		t.Fatalf("runAppsShow: %v", err)
	}

	if !bytes.Contains(buf.Bytes(), []byte("myapp")) {
		t.Errorf("expected output to contain myapp, got %q", buf.String())
	}
}

func TestAppsActivate(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer ts.Close()

	SetClient(client.NewWithHTTP(ts.URL, "token", ts.Client()))

	buf := new(bytes.Buffer)
	output.SetOut(buf)
	appsActivateCmd.SetOut(buf)
	appsActivateCmd.Flags().Set("name", "myapp")

	if err := runAppsActivate(appsActivateCmd, nil); err != nil {
		t.Fatalf("runAppsActivate: %v", err)
	}

	if !bytes.Contains(buf.Bytes(), []byte("active")) {
		t.Errorf("expected output to contain active, got %q", buf.String())
	}
}
