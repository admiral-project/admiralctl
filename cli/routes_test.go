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

func TestRoutesListCmd(t *testing.T) {
	httpClient := &http.Client{
		Transport: mockRoundTripper(func(r *http.Request) (*http.Response, error) {
			routes := []map[string]interface{}{
				{
					"hostname":        "wiki.example.com",
					"route_kind":      "http",
					"app_instance_id": "inst-1",
					"service_name":    "main",
					"target_url":      "http://10.0.0.1:80",
					"status":          "active",
				},
			}
			body, _ := json.Marshal(routes)
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
		err := runRoutesList(routesListCmd, nil)
		if err != nil {
			t.Fatalf("runRoutesList failed: %v", err)
		}
	})

	if !strings.Contains(got, "wiki.example.com") || !strings.Contains(got, "inst-1") {
		t.Fatalf("expected wiki.example.com and inst-1 in output, got %q", got)
	}
}

func TestRoutesSyncCmd(t *testing.T) {
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
	routesSyncCmd.SetOut(&out)

	err := runRoutesSync(routesSyncCmd, nil)
	if err != nil {
		t.Fatalf("runRoutesSync failed: %v", err)
	}

	got := out.String()
	if !strings.Contains(got, "Routes synchronized successfully") {
		t.Fatalf("unexpected output: %q", got)
	}
}

func TestRoutesShowCmd(t *testing.T) {
	httpClient := &http.Client{
		Transport: mockRoundTripper(func(r *http.Request) (*http.Response, error) {
			route := map[string]interface{}{"hostname": "route-show.example.com"}
			body, _ := json.Marshal(route)
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
	routesShowCmd.SetOut(&out)

	err := runRoutesShow(routesShowCmd, []string{"route-show.example.com"})
	if err != nil {
		t.Fatalf("runRoutesShow failed: %v", err)
	}

	got := out.String()
	if !strings.Contains(got, "\"hostname\": \"route-show.example.com\"") {
		t.Fatalf("unexpected output: %q", got)
	}
}

func TestRoutesEnableCmd(t *testing.T) {
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
	routesEnableCmd.SetOut(&out)

	err := runRoutesEnable(routesEnableCmd, []string{"route-enable.example.com"})
	if err != nil {
		t.Fatalf("runRoutesEnable failed: %v", err)
	}

	got := out.String()
	if !strings.Contains(got, "Route route-enable.example.com successfully") {
		t.Fatalf("unexpected output: %q", got)
	}
}

func TestRoutesDisableCmd(t *testing.T) {
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
	routesDisableCmd.SetOut(&out)
	_ = routesDisableCmd.Flags().Set("force", "true")

	err := runRoutesDisable(routesDisableCmd, []string{"route-disable.example.com"})
	if err != nil {
		t.Fatalf("runRoutesDisable failed: %v", err)
	}

	got := out.String()
	if !strings.Contains(got, "Route route-disable.example.com successfully") {
		t.Fatalf("unexpected output: %q", got)
	}
}
