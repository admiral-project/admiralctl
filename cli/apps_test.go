// SPDX-FileCopyrightText: William Moreno Reyes CP | MBA
// SPDX-License-Identifier: Apache-2.0

package cli

import (
	"bytes"
	"io"
	"net/http"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestReadAndValidateERPNextExampleAppFile(t *testing.T) {
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	examplePath := filepath.Join(filepath.Dir(file), "testdata", "erpnext.yaml")

	_, payload, err := readAndValidateAppFile(nil, examplePath)
	if err != nil {
		t.Fatalf("readAndValidateAppFile: %v", err)
	}
	if payload.Name != "erpnext" {
		t.Fatalf("expected erpnext payload, got %q", payload.Name)
	}
}

func TestReadAndValidateAppFileRejectsTraversal(t *testing.T) {
	_, _, err := readAndValidateAppFile(nil, "../testdata/app.yaml")
	if err == nil {
		t.Fatal("expected path traversal to be rejected")
	}
	if !strings.Contains(err.Error(), "validate file path") {
		t.Fatalf("expected validation error, got %v", err)
	}
}

func TestAppsApplyCmd(t *testing.T) {
	httpClient := &http.Client{
		Transport: mockRoundTripper(func(r *http.Request) (*http.Response, error) {
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(strings.NewReader(`{"name": "erpnext"}`)),
				Header:     make(http.Header),
			}, nil
		}),
	}

	c := newMockClient(t, httpClient)
	SetClient(c)

	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	examplePath := filepath.Join(filepath.Dir(file), "testdata", "erpnext.yaml")

	var out bytes.Buffer
	appsApplyCmd.SetOut(&out)
	_ = appsApplyCmd.Flags().Set("file", examplePath)

	err := runAppsApply(appsApplyCmd, nil)
	if err != nil {
		t.Fatalf("runAppsApply failed: %v", err)
	}

	got := out.String()
	if !strings.Contains(got, "applied successfully") || !strings.Contains(got, "erpnext") {
		t.Fatalf("unexpected output: %q", got)
	}
}

func TestAppsValidateCmd(t *testing.T) {
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	examplePath := filepath.Join(filepath.Dir(file), "testdata", "erpnext.yaml")

	var out bytes.Buffer
	appsValidateCmd.SetOut(&out)
	_ = appsValidateCmd.Flags().Set("file", examplePath)

	err := runAppsValidate(appsValidateCmd, nil)
	if err != nil {
		t.Fatalf("runAppsValidate failed: %v", err)
	}

	got := out.String()
	if !strings.Contains(got, "YAML Validation: PASSED") {
		t.Fatalf("unexpected output: %q", got)
	}
}

func TestAppsActivateCmd(t *testing.T) {
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
	appsActivateCmd.SetOut(&out)
	_ = appsActivateCmd.Flags().Set("name", "erpnext")

	err := runAppsActivate(appsActivateCmd, nil)
	if err != nil {
		t.Fatalf("runAppsActivate failed: %v", err)
	}

	got := out.String()
	if !strings.Contains(got, "is now active") {
		t.Fatalf("unexpected output: %q", got)
	}
}

func TestAppsDeactivateCmd(t *testing.T) {
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
	appsDeactivateCmd.SetOut(&out)
	_ = appsDeactivateCmd.Flags().Set("name", "erpnext")
	_ = appsDeactivateCmd.Flags().Set("force", "true")

	err := runAppsDeactivate(appsDeactivateCmd, nil)
	if err != nil {
		t.Fatalf("runAppsDeactivate failed: %v", err)
	}

	got := out.String()
	if !strings.Contains(got, "is now inactive") {
		t.Fatalf("unexpected output: %q", got)
	}
}
