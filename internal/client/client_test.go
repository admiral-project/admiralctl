// SPDX-FileCopyrightText: William Moreno Reyes CP | MBA
// SPDX-License-Identifier: Apache-2.0

package client

import (
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/admiral-project/admiral/admirald/pkg/admiral"
)

type roundTripperFunc func(*http.Request) (*http.Response, error)

func (f roundTripperFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

func newTestHTTPClient(handler func(*http.Request) (*http.Response, error)) *http.Client {
	return &http.Client{Transport: roundTripperFunc(handler)}
}

func jsonResponse(status int, payload any) (*http.Response, error) {
	var body strings.Builder
	if payload != nil {
		if err := json.NewEncoder(&body).Encode(payload); err != nil {
			return nil, err
		}
	}
	return &http.Response{
		StatusCode: status,
		Body:       io.NopCloser(strings.NewReader(body.String())),
		Header:     make(http.Header),
	}, nil
}

func TestRetryOnServerError(t *testing.T) {
	failCount := 0
	client := newTestHTTPClient(func(r *http.Request) (*http.Response, error) {
		failCount++
		if failCount <= 2 {
			return jsonResponse(http.StatusInternalServerError, nil)
		}
		return jsonResponse(http.StatusOK, map[string]interface{}{"status": "ok"})
	})

	c := &Client{
		serverURL:  "https://example.com",
		token:      "token",
		http:       client,
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
	client := newTestHTTPClient(func(r *http.Request) (*http.Response, error) {
		attempts++
		return jsonResponse(http.StatusBadRequest, nil)
	})

	c := &Client{
		serverURL:  "https://example.com",
		token:      "token",
		http:       client,
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
	client := newTestHTTPClient(func(r *http.Request) (*http.Response, error) {
		attempts++
		return jsonResponse(http.StatusServiceUnavailable, nil)
	})

	c := &Client{
		serverURL:  "https://example.com",
		token:      "token",
		http:       client,
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
	client := newTestHTTPClient(func(r *http.Request) (*http.Response, error) {
		seen = append(seen, r.Method+" "+r.URL.Path)
		switch r.URL.Path {
		case "/api/v1/routes":
			switch r.Method {
			case http.MethodGet:
				return jsonResponse(http.StatusOK, []map[string]interface{}{{"hostname": "wiki123456.apps.example.com"}})
			case http.MethodPost:
				return jsonResponse(http.StatusOK, nil)
			default:
				return jsonResponse(http.StatusMethodNotAllowed, nil)
			}
		case "/api/v1/routes/wiki123456.apps.example.com":
			return jsonResponse(http.StatusOK, map[string]interface{}{"hostname": "wiki123456.apps.example.com"})
		case "/api/v1/routes/wiki123456.apps.example.com/enable":
			return jsonResponse(http.StatusOK, nil)
		case "/api/v1/routes/wiki123456.apps.example.com/disable":
			return jsonResponse(http.StatusOK, nil)
		default:
			return jsonResponse(http.StatusNotFound, nil)
		}
	})

	c := &Client{serverURL: "https://example.com", token: "token", http: client}

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

func TestGetBackupStorageConfig(t *testing.T) {
	client := newTestHTTPClient(func(r *http.Request) (*http.Response, error) {
		if r.Method != http.MethodGet || r.URL.Path != "/api/admin/settings/backup-storage" {
			return jsonResponse(http.StatusNotFound, nil)
		}
		return jsonResponse(http.StatusOK, admiral.BackupStorageConfig{
			Backend:  "s3",
			Bucket:    "admiral-backups",
			Region:    "us-east-1",
			Endpoint:  "https://s3.example.com",
			AccessKeyEnv: "AKID123",
		})
	})

	c := &Client{serverURL: "https://example.com", token: "token", http: client}
	cfg, err := c.GetBackupStorageConfig()
	if err != nil {
		t.Fatalf("get backup storage config: %v", err)
	}
	if cfg.Backend != "s3" || cfg.Bucket != "admiral-backups" {
		t.Fatalf("unexpected config: %+v", cfg)
	}
}

func TestSetBackupStorageConfig(t *testing.T) {
	client := newTestHTTPClient(func(r *http.Request) (*http.Response, error) {
		if r.Method != http.MethodPut || r.URL.Path != "/api/admin/settings/backup-storage" {
			return jsonResponse(http.StatusNotFound, nil)
		}
		return jsonResponse(http.StatusOK, nil)
	})

	c := &Client{serverURL: "https://example.com", token: "token", http: client}
		err := c.SetBackupStorageConfig(admiral.BackupStorageConfig{
		Backend:  "s3",
		Bucket:    "admiral-backups",
		Region:    "us-east-1",
		Endpoint:  "https://s3.example.com",
		AccessKeyEnv: "AKID123",
		SecretKeyEnv: "sk-secret",
	})
	if err != nil {
		t.Fatalf("set backup storage config: %v", err)
	}
}

func TestTestBackupStorageConfig(t *testing.T) {
	client := newTestHTTPClient(func(r *http.Request) (*http.Response, error) {
		if r.Method != http.MethodPost || r.URL.Path != "/api/admin/settings/backup-storage/test" {
			return jsonResponse(http.StatusNotFound, nil)
		}
		return jsonResponse(http.StatusOK, nil)
	})

	c := &Client{serverURL: "https://example.com", token: "token", http: client}
	if err := c.TestBackupStorageConfig(); err != nil {
		t.Fatalf("test backup storage config: %v", err)
	}
}

func TestDeleteBackup(t *testing.T) {
	client := newTestHTTPClient(func(r *http.Request) (*http.Response, error) {
		if r.Method != http.MethodDelete || r.URL.Path != "/api/admin/backups/bk_001" {
			return jsonResponse(http.StatusNotFound, nil)
		}
		return jsonResponse(http.StatusOK, nil)
	})

	c := &Client{serverURL: "https://example.com", token: "token", http: client}
	if err := c.DeleteBackup("bk_001"); err != nil {
		t.Fatalf("delete backup: %v", err)
	}
}

func TestDeleteBackupAccepts202(t *testing.T) {
	client := newTestHTTPClient(func(r *http.Request) (*http.Response, error) {
		return jsonResponse(http.StatusAccepted, nil)
	})

	c := &Client{serverURL: "https://example.com", token: "token", http: client}
	if err := c.DeleteBackup("bk_001"); err != nil {
		t.Fatalf("delete backup should accept 202: %v", err)
	}
}

func TestDeleteBackupError(t *testing.T) {
	client := newTestHTTPClient(func(r *http.Request) (*http.Response, error) {
		return jsonResponse(http.StatusNotFound, map[string]string{"error": "backup not found"})
	})

	c := &Client{serverURL: "https://example.com", token: "token", http: client}
	err := c.DeleteBackup("bk_001")
	if err == nil {
		t.Fatal("expected error for backup not found")
	}
}

func TestPruneBackups(t *testing.T) {
	client := newTestHTTPClient(func(r *http.Request) (*http.Response, error) {
		if r.Method != http.MethodPost || r.URL.Path != "/api/admin/backups/prune" {
			return jsonResponse(http.StatusNotFound, nil)
		}
		return jsonResponse(http.StatusOK, nil)
	})

	c := &Client{serverURL: "https://example.com", token: "token", http: client}
	if err := c.PruneBackups(); err != nil {
		t.Fatalf("prune backups: %v", err)
	}
}

func TestMigrateInstance(t *testing.T) {
	client := newTestHTTPClient(func(r *http.Request) (*http.Response, error) {
		if r.Method != http.MethodPost || r.URL.Path != "/api/admin/instances/inst_001/migrate" {
			return jsonResponse(http.StatusNotFound, nil)
		}
		return jsonResponse(http.StatusAccepted, admiral.MigrateAppResponse{
			OperationID:       "op_001",
			InstanceID:        "inst_001",
			LogicalInstanceID: "li_001",
			Status:            "accepted",
		})
	})

	c := &Client{serverURL: "https://example.com", token: "token", http: client}
	res, err := c.MigrateInstance("inst_001", "worker-02")
	if err != nil {
		t.Fatalf("migrate instance: %v", err)
	}
	if res.OperationID != "op_001" || res.InstanceID != "inst_001" || res.Status != "accepted" {
		t.Fatalf("unexpected migrate response: %+v", res)
	}
}

func TestMigrateInstanceError(t *testing.T) {
	client := newTestHTTPClient(func(r *http.Request) (*http.Response, error) {
		return jsonResponse(http.StatusBadRequest, map[string]string{"error": "node not found"})
	})

	c := &Client{serverURL: "https://example.com", token: "token", http: client}
	_, err := c.MigrateInstance("inst_001", "nonexistent")
	if err == nil {
		t.Fatal("expected error for bad migrate request")
	}
}
