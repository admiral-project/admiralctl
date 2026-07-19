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

	"github.com/admiral-project/admiral/admiralctl/internal/output"
	"github.com/admiral-project/admiral/admirald/pkg/admiral"
	"github.com/spf13/cobra"
)

func TestPrintProvisionAccessData(t *testing.T) {
	cmd := &cobra.Command{}
	var out bytes.Buffer
	cmd.SetOut(&out)

	printProvisionAccessData(cmd, []admiral.Credential{
		{Service: "backend", Name: "ADMIN_PASSWORD", Value: "secret", Generate: "password"},
		{Service: "backend", Name: "Usuario administrador", Value: "Administrator", Kind: "notice"},
	})

	output := out.String()
	if !strings.Contains(output, "Initial credentials:") {
		t.Fatalf("expected credentials heading, got %q", output)
	}
	if !strings.Contains(output, "backend.ADMIN_PASSWORD: secret") {
		t.Fatalf("expected credential value, got %q", output)
	}
	if !strings.Contains(output, "Usuario administrador: Administrator") {
		t.Fatalf("expected setup notice, got %q", output)
	}
}

func TestQuietProvisionJSONOmitsCredentials(t *testing.T) {
	res := admiral.ProvisionResponse{
		OperationID: "op-1",
		Status:      "queued",
		Credentials: []admiral.Credential{
			{Service: "web", Name: "ADMIN_PASSWORD", Value: "secret"},
		},
	}

	res = provisionResponseForOutput(res, true)
	var out bytes.Buffer
	output.PrintJSON(&out, res)

	if strings.Contains(out.String(), "secret") || strings.Contains(out.String(), "credentials") {
		t.Fatalf("quiet JSON exposed credentials: %q", out.String())
	}
}

func TestInstancesCredentialsCmd(t *testing.T) {
	httpClient := &http.Client{
		Transport: mockRoundTripper(func(r *http.Request) (*http.Response, error) {
			creds := []admiral.Credential{
				{Service: "main", Name: "password", Value: "pwd123"},
			}
			body, _ := json.Marshal(creds)
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
	instancesCredentialsCmd.SetOut(&out)

	err := runInstancesCredentials(instancesCredentialsCmd, []string{"inst-creds"})
	if err != nil {
		t.Fatalf("runInstancesCredentials failed: %v", err)
	}

	got := out.String()
	if !strings.Contains(got, "main.password: pwd123") {
		t.Fatalf("unexpected output: %q", got)
	}
}

func TestInstancesInspectCmd(t *testing.T) {
	httpClient := &http.Client{
		Transport: mockRoundTripper(func(r *http.Request) (*http.Response, error) {
			if r.Method == "POST" {
				res := admiral.OperationResponse{OperationID: "op-inspect"}
				body, _ := json.Marshal(res)
				return &http.Response{
					StatusCode: http.StatusAccepted,
					Body:       io.NopCloser(bytes.NewReader(body)),
					Header:     make(http.Header),
				}, nil
			}
			res := map[string]interface{}{"result": "inspect-ok"}
			body, _ := json.Marshal(res)
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
	instancesInspectCmd.SetOut(&out)
	_ = instancesInspectCmd.Flags().Set("result", "false")

	err := runInstancesInspect(instancesInspectCmd, []string{"inst-inspect"})
	if err != nil {
		t.Fatalf("runInstancesInspect failed: %v", err)
	}
	got := out.String()
	if !strings.Contains(got, "Inspect task queued") || !strings.Contains(got, "op-inspect") {
		t.Fatalf("unexpected output: %q", got)
	}

	out.Reset()
	_ = instancesInspectCmd.Flags().Set("result", "true")
	err = runInstancesInspect(instancesInspectCmd, []string{"inst-inspect"})
	if err != nil {
		t.Fatalf("runInstancesInspect --result failed: %v", err)
	}
	got = out.String()
	if !strings.Contains(got, "inspect-ok") {
		t.Fatalf("unexpected output: %q", got)
	}
}

func TestInstancesProvisionCmd(t *testing.T) {
	httpClient := &http.Client{
		Transport: mockRoundTripper(func(r *http.Request) (*http.Response, error) {
			res := admiral.ProvisionResponse{
				OperationID: "op-provision",
				Credentials: []admiral.Credential{
					{Service: "main", Name: "password", Value: "pass1"},
				},
			}
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
	instancesProvisionCmd.SetOut(&out)
	_ = instancesProvisionCmd.Flags().Set("app", "erpnext")
	_ = instancesProvisionCmd.Flags().Set("tier", "small")
	_ = instancesProvisionCmd.Flags().Set("customer", "cust-1")

	err := runInstancesProvision(instancesProvisionCmd, nil)
	if err != nil {
		t.Fatalf("runInstancesProvision failed: %v", err)
	}

	got := out.String()
	if !strings.Contains(got, "Provisioning queued successfully") || !strings.Contains(got, "op-provision") {
		t.Fatalf("unexpected output: %q", got)
	}
}

func TestInstancesActionCmd(t *testing.T) {
	httpClient := &http.Client{
		Transport: mockRoundTripper(func(r *http.Request) (*http.Response, error) {
			res := admiral.OperationResponse{OperationID: "op-action"}
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
	instancesPauseCmd.SetOut(&out)

	err := runInstancesAction("pause")(instancesPauseCmd, []string{"inst-pause"})
	if err != nil {
		t.Fatalf("runInstancesAction failed: %v", err)
	}

	got := out.String()
	if !strings.Contains(got, "Action pause queued successfully") || !strings.Contains(got, "op-action") {
		t.Fatalf("unexpected output: %q", got)
	}
}

func TestInstancesRestartCmd(t *testing.T) {
	httpClient := &http.Client{
		Transport: mockRoundTripper(func(r *http.Request) (*http.Response, error) {
			res := admiral.OperationResponse{OperationID: "op-restart"}
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
	instancesRestartCmd.SetOut(&out)
	_ = instancesRestartCmd.Flags().Set("force", "true")

	err := runInstancesRestart(instancesRestartCmd, []string{"inst-restart"})
	if err != nil {
		t.Fatalf("runInstancesRestart failed: %v", err)
	}

	got := out.String()
	if !strings.Contains(got, "restarted") || !strings.Contains(got, "op-restart") {
		t.Fatalf("unexpected output: %q", got)
	}
}

func TestInstancesBackupCmd(t *testing.T) {
	httpClient := &http.Client{
		Transport: mockRoundTripper(func(r *http.Request) (*http.Response, error) {
			res := admiral.OperationResponse{OperationID: "op-backup"}
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
	instancesBackupCmd.SetOut(&out)
	_ = instancesBackupCmd.Flags().Set("service", "db")

	err := runInstancesBackup(instancesBackupCmd, []string{"inst-backup"})
	if err != nil {
		t.Fatalf("runInstancesBackup failed: %v", err)
	}

	got := out.String()
	if !strings.Contains(got, "Action backup queued successfully") || !strings.Contains(got, "op-backup") {
		t.Fatalf("unexpected output: %q", got)
	}
}

func TestInstancesResizeCmd(t *testing.T) {
	httpClient := &http.Client{
		Transport: mockRoundTripper(func(r *http.Request) (*http.Response, error) {
			res := admiral.OperationResponse{OperationID: "op-resize"}
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
	instancesResizeCmd.SetOut(&out)
	_ = instancesResizeCmd.Flags().Set("tier", "large")

	withMockStdin(t, "y\n", func() {
		err := runInstancesResize(instancesResizeCmd, []string{"inst-resize"})
		if err != nil {
			t.Fatalf("runInstancesResize failed: %v", err)
		}
	})

	got := out.String()
	if !strings.Contains(got, "Resize queued successfully") || !strings.Contains(got, "op-resize") {
		t.Fatalf("unexpected output: %q", got)
	}
}

func TestInstancesMigrateCmd(t *testing.T) {
	httpClient := &http.Client{
		Transport: mockRoundTripper(func(r *http.Request) (*http.Response, error) {
			res := admiral.MigrateAppResponse{
				OperationID:       "op-migrate",
				InstanceID:        "inst-migrate",
				LogicalInstanceID: "li-1",
				Status:            "accepted",
			}
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
	instancesMigrateCmd.SetOut(&out)
	_ = instancesMigrateCmd.Flags().Set("target-node", "worker-02")

	withMockStdin(t, "y\n", func() {
		err := runInstancesMigrate(instancesMigrateCmd, []string{"inst-migrate"})
		if err != nil {
			t.Fatalf("runInstancesMigrate failed: %v", err)
		}
	})

	got := out.String()
	if !strings.Contains(got, "Migration started") || !strings.Contains(got, "op-migrate") {
		t.Fatalf("unexpected output: %q", got)
	}
}
