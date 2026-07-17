// SPDX-FileCopyrightText: William Moreno Reyes CP | MBA
// SPDX-License-Identifier: Apache-2.0

package cli

import (
	"bytes"
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
