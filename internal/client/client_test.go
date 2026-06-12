// SPDX-FileCopyrightText: William Moreno Reyes CP | MBA
// SPDX-License-Identifier: Apache-2.0

package client

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestRetryOnServerError(t *testing.T) {
	failCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		failCount++
		if failCount <= 2 {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]interface{}{"status": "ok"})
	}))
	defer server.Close()

	c := &Client{
		serverURL:  server.URL,
		token:      "token",
		http:       server.Client(),
		maxRetries: 3,
		retryDelay: 1 * time.Millisecond,
	}

	res, err := c.GetStatus()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res["status"] != "ok" {
		t.Fatalf("unexpected response: %#v", res)
	}
	if failCount != 3 {
		t.Fatalf("expected 3 attempts (2 fails + 1 success), got %d", failCount)
	}
}

func TestNoRetryOnClientError(t *testing.T) {
	attempts := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts++
		w.WriteHeader(http.StatusBadRequest)
	}))
	defer server.Close()

	c := &Client{
		serverURL:  server.URL,
		token:      "token",
		http:       server.Client(),
		maxRetries: 3,
		retryDelay: 1 * time.Millisecond,
	}

	_, err := c.GetStatus()
	if err == nil {
		t.Fatal("expected error for HTTP 400")
	}
	if attempts != 1 {
		t.Fatalf("expected 1 attempt (no retry on 4xx), got %d", attempts)
	}
}

func TestRetryExhaustion(t *testing.T) {
	attempts := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts++
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	defer server.Close()

	c := &Client{
		serverURL:  server.URL,
		token:      "token",
		http:       server.Client(),
		maxRetries: 2,
		retryDelay: 1 * time.Millisecond,
	}

	_, err := c.GetStatus()
	if err == nil {
		t.Fatal("expected error after exhausting retries")
	}
	if attempts != 3 {
		t.Fatalf("expected 3 attempts (1 initial + 2 retries), got %d", attempts)
	}
}

func TestBackoffDuration(t *testing.T) {
	c := &Client{retryDelay: 1 * time.Second}
	prev := time.Duration(0)
	for attempt := 1; attempt <= 3; attempt++ {
		d := c.backoff(attempt)
		if d <= prev {
			t.Fatalf("backoff attempt %d = %v should be > %v", attempt, d, prev)
		}
		prev = d
	}
}

func TestRoutesAPI(t *testing.T) {
	var seen []string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		seen = append(seen, r.Method+" "+r.URL.Path)
		switch r.URL.Path {
		case "/api/v1/routes":
			switch r.Method {
			case http.MethodGet:
				_ = json.NewEncoder(w).Encode([]map[string]interface{}{{"hostname": "wiki123456.apps.example.com"}})
			case http.MethodPost:
				w.WriteHeader(http.StatusOK)
			default:
				w.WriteHeader(http.StatusMethodNotAllowed)
			}
		case "/api/v1/routes/wiki123456.apps.example.com":
			_ = json.NewEncoder(w).Encode(map[string]interface{}{"hostname": "wiki123456.apps.example.com"})
		case "/api/v1/routes/wiki123456.apps.example.com/enable":
			w.WriteHeader(http.StatusOK)
		case "/api/v1/routes/wiki123456.apps.example.com/disable":
			w.WriteHeader(http.StatusOK)
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	c := &Client{serverURL: server.URL, token: "token", http: server.Client()}

	routes, err := c.GetRoutes()
	if err != nil {
		t.Fatalf("get routes: %v", err)
	}
	if len(routes) != 1 || routes[0]["hostname"] != "wiki123456.apps.example.com" {
		t.Fatalf("unexpected routes payload: %#v", routes)
	}
	if _, err := c.GetRoute("wiki123456.apps.example.com"); err != nil {
		t.Fatalf("get route: %v", err)
	}
	if err := c.EnableRoute("wiki123456.apps.example.com"); err != nil {
		t.Fatalf("enable route: %v", err)
	}
	if err := c.DisableRoute("wiki123456.apps.example.com"); err != nil {
		t.Fatalf("disable route: %v", err)
	}
	if err := c.SyncRoutes(); err != nil {
		t.Fatalf("sync routes: %v", err)
	}

	want := []string{
		"GET /api/v1/routes",
		"GET /api/v1/routes/wiki123456.apps.example.com",
		"POST /api/v1/routes/wiki123456.apps.example.com/enable",
		"POST /api/v1/routes/wiki123456.apps.example.com/disable",
		"POST /api/v1/routes",
	}
	if len(seen) != len(want) {
		t.Fatalf("unexpected request count %d, want %d", len(seen), len(want))
	}
	for i := range want {
		if seen[i] != want[i] {
			t.Fatalf("request %d = %q, want %q", i, seen[i], want[i])
		}
	}
}

func TestFormatHTTPErrorUsesStructuredMessage(t *testing.T) {
	err := formatHTTPError("fetch nodes", http.StatusBadRequest, []byte(`{"error":"token expired"}`))
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "fetch nodes failed: HTTP 400 - token expired") {
		t.Fatalf("unexpected error message: %v", err)
	}
}

func TestFormatHTTPErrorRedactsUnstructuredBody(t *testing.T) {
	err := formatHTTPError("fetch nodes", http.StatusBadGateway, []byte("stacktrace: secret-token"))
	if err == nil {
		t.Fatal("expected error")
	}
	if strings.Contains(err.Error(), "secret-token") {
		t.Fatalf("expected unstructured body to be redacted, got %v", err)
	}
	if !strings.Contains(err.Error(), "server returned an unstructured error response") {
		t.Fatalf("unexpected error message: %v", err)
	}
}
