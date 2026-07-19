// SPDX-FileCopyrightText: William Moreno Reyes CP | MBA
// SPDX-License-Identifier: Apache-2.0

package client

import (
	"encoding/json"
	"errors"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/admiral-project/admiral/admirald/pkg/admiral"
)

func TestProvisionRejectedErrorFormatting(t *testing.T) {
	err := (&ProvisionRejectedError{Response: admiral.ProvisioningRejectedResponse{
		Message:     "blocked by policy",
		OperationID: "op_123",
	}}).Error()

	if err != "blocked by policy (operation_id=op_123)" {
		t.Fatalf("unexpected error string %q", err)
	}
}

func TestProvisionRejectedErrorFallbacks(t *testing.T) {
	tests := []struct {
		name string
		resp admiral.ProvisioningRejectedResponse
		want string
	}{
		{
			name: "uses error field",
			resp: admiral.ProvisioningRejectedResponse{Error: "temporary block"},
			want: "temporary block",
		},
		{
			name: "uses generic fallback",
			resp: admiral.ProvisioningRejectedResponse{},
			want: "provisioning rejected by policy",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := (&ProvisionRejectedError{Response: tt.resp}).Error(); got != tt.want {
				t.Fatalf("unexpected error string %q want %q", got, tt.want)
			}
		})
	}
}

func TestParsePolicyRejectedError(t *testing.T) {
	body := []byte(`{"code":"no_capacity","message":"No nodes available","operation_id":"op_456"}`)
	err := parsePolicyRejectedError(http.StatusServiceUnavailable, body)
	if err == nil {
		t.Fatal("expected rejected error")
	}

	rejected, ok := err.(*ProvisionRejectedError)
	if !ok {
		t.Fatalf("expected ProvisionRejectedError, got %T", err)
	}
	if rejected.Response.Code != "no_capacity" || rejected.Response.OperationID != "op_456" {
		t.Fatalf("unexpected rejected response %+v", rejected.Response)
	}
}

func TestParsePolicyRejectedErrorIgnoresOtherStatuses(t *testing.T) {
	if err := parsePolicyRejectedError(http.StatusBadRequest, []byte(`{"code":"x"}`)); err != nil {
		t.Fatalf("expected nil for non-503 status, got %v", err)
	}
}

func TestSanitizeErrorBodyUsesMessageField(t *testing.T) {
	got := sanitizeErrorBody([]byte(`{"message":"human readable failure"}`))
	if got != "human readable failure" {
		t.Fatalf("unexpected sanitized body %q", got)
	}
}

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
	client := newTestHTTPClient(func(_ *http.Request) (*http.Response, error) {
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

func TestDoRequestRejectsOversizedResponse(t *testing.T) {
	c := &Client{
		serverURL: "https://example.com",
		token:     "token",
		http: newTestHTTPClient(func(_ *http.Request) (*http.Response, error) {
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(strings.NewReader(strings.Repeat("x", maxResponseSize+1))),
				Header:     make(http.Header),
			}, nil
		}),
	}

	_, _, err := c.doRequest(http.MethodGet, "https://example.com/status", nil)
	if err == nil || !strings.Contains(err.Error(), "response body exceeds") {
		t.Fatalf("expected oversized response error, got %v", err)
	}
}

func TestNoRetryOnClientError(t *testing.T) {
	attempts := 0
	client := newTestHTTPClient(func(_ *http.Request) (*http.Response, error) {
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
	client := newTestHTTPClient(func(_ *http.Request) (*http.Response, error) {
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
			Backend:      "s3",
			Bucket:       "admiral-backups",
			Region:       "us-east-1",
			Endpoint:     "https://s3.example.com",
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
		Backend:      "s3",
		Bucket:       "admiral-backups",
		Region:       "us-east-1",
		Endpoint:     "https://s3.example.com",
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
	client := newTestHTTPClient(func(_ *http.Request) (*http.Response, error) {
		return jsonResponse(http.StatusAccepted, nil)
	})

	c := &Client{serverURL: "https://example.com", token: "token", http: client}
	if err := c.DeleteBackup("bk_001"); err != nil {
		t.Fatalf("delete backup should accept 202: %v", err)
	}
}

func TestDeleteBackupError(t *testing.T) {
	client := newTestHTTPClient(func(_ *http.Request) (*http.Response, error) {
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
	client := newTestHTTPClient(func(_ *http.Request) (*http.Response, error) {
		return jsonResponse(http.StatusBadRequest, map[string]string{"error": "node not found"})
	})

	c := &Client{serverURL: "https://example.com", token: "token", http: client}
	_, err := c.MigrateInstance("inst_001", "nonexistent")
	if err == nil {
		t.Fatal("expected error for bad migrate request")
	}
}

func TestOptions(t *testing.T) {
	c := &Client{http: &http.Client{}}
	WithTimeout(10 * time.Second)(c)
	if c.http.Timeout != 10*time.Second {
		t.Errorf("WithTimeout failed, got %v", c.http.Timeout)
	}
	WithRetries(5, 2*time.Second)(c)
	if c.maxRetries != 5 || c.retryDelay != 2*time.Second {
		t.Errorf("WithRetries failed, got %d, %v", c.maxRetries, c.retryDelay)
	}
	WithOperator("jules")(c)
	if c.operator != "jules" {
		t.Errorf("WithOperator failed, got %s", c.operator)
	}
}

func TestNodeManagement(t *testing.T) {
	client := newTestHTTPClient(func(r *http.Request) (*http.Response, error) {
		switch r.URL.Path {
		case "/api/v1/nodes":
			if r.Method == "POST" {
				return jsonResponse(http.StatusOK, nil)
			}
			return jsonResponse(http.StatusOK, []map[string]interface{}{{"id": "node-1"}})
		case "/api/v1/nodes/node-1":
			if r.Method == "DELETE" {
				return jsonResponse(http.StatusOK, nil)
			}
			return jsonResponse(http.StatusOK, map[string]interface{}{"id": "node-1"})
		case "/api/v1/nodes/node-1/enable":
			return jsonResponse(http.StatusOK, nil)
		case "/api/v1/nodes/node-1/disable":
			return jsonResponse(http.StatusOK, nil)
		case "/api/v1/nodes/node-1/ready":
			return jsonResponse(http.StatusOK, map[string]interface{}{"ready": true})
		}
		return jsonResponse(http.StatusNotFound, nil)
	})
	c := &Client{serverURL: "https://example.com", token: "token", http: client}

	if err := c.RegisterNode(admiral.RegisterNodeRequest{}); err != nil {
		t.Errorf("RegisterNode failed: %v", err)
	}
	if _, err := c.GetNodes(); err != nil {
		t.Errorf("GetNodes failed: %v", err)
	}
	if _, err := c.GetNode("node-1"); err != nil {
		t.Errorf("GetNode failed: %v", err)
	}
	if err := c.EnableNode("node-1"); err != nil {
		t.Errorf("EnableNode failed: %v", err)
	}
	if err := c.DisableNode("node-1"); err != nil {
		t.Errorf("DisableNode failed: %v", err)
	}
	if _, err := c.NodeReady("node-1"); err != nil {
		t.Errorf("NodeReady failed: %v", err)
	}
	if err := c.RemoveNode("node-1"); err != nil {
		t.Errorf("RemoveNode failed: %v", err)
	}
}

func TestAppManagement(t *testing.T) {
	client := newTestHTTPClient(func(r *http.Request) (*http.Response, error) {
		switch r.URL.Path {
		case "/api/v1/apps":
			if r.Method == "POST" {
				return jsonResponse(http.StatusOK, map[string]interface{}{"name": "my-app"})
			}
			return jsonResponse(http.StatusOK, []map[string]interface{}{{"name": "my-app"}})
		case "/api/v1/apps/my-app":
			return jsonResponse(http.StatusOK, map[string]interface{}{"name": "my-app"})
		case "/api/v1/apps/my-app/status":
			return jsonResponse(http.StatusOK, nil)
		}
		return jsonResponse(http.StatusNotFound, nil)
	})
	c := &Client{serverURL: "https://example.com", token: "token", http: client}

	if _, err := c.ApplyApp("yaml"); err != nil {
		t.Errorf("ApplyApp failed: %v", err)
	}
	if _, err := c.GetApps(); err != nil {
		t.Errorf("GetApps failed: %v", err)
	}
	if _, err := c.GetApp("my-app"); err != nil {
		t.Errorf("GetApp failed: %v", err)
	}
	if err := c.UpdateAppStatus("my-app", "active"); err != nil {
		t.Errorf("UpdateAppStatus failed: %v", err)
	}
}

func TestProvisioningAndActions(t *testing.T) {
	client := newTestHTTPClient(func(r *http.Request) (*http.Response, error) {
		switch r.URL.Path {
		case "/api/v1/instances":
			return jsonResponse(http.StatusOK, []map[string]interface{}{{"id": "inst-1"}})
		case "/api/v1/customer-apps":
			if r.Method == "POST" {
				return jsonResponse(http.StatusAccepted, admiral.ProvisionResponse{OperationID: "op-1"})
			}
			return jsonResponse(http.StatusOK, []map[string]interface{}{{"id": "inst-1"}})
		case "/api/v1/customer-apps/action":
			return jsonResponse(http.StatusAccepted, admiral.OperationResponse{OperationID: "op-2"})
		case "/api/v1/customer-apps/inst-1":
			return jsonResponse(http.StatusOK, map[string]interface{}{"id": "inst-1"})
		case "/api/v1/customer-apps/inst-1/credentials":
			return jsonResponse(http.StatusOK, []admiral.Credential{})
		}
		return jsonResponse(http.StatusNotFound, nil)
	})
	c := &Client{serverURL: "https://example.com", token: "token", http: client}

	if _, err := c.ProvisionApp(admiral.ProvisionRequest{}); err != nil {
		t.Errorf("ProvisionApp failed: %v", err)
	}
	if _, err := c.TriggerAction("inst-1", "pause"); err != nil {
		t.Errorf("TriggerAction failed: %v", err)
	}
	if _, err := c.TriggerActionWithService("inst-1", "backup", "db"); err != nil {
		t.Errorf("TriggerActionWithService failed: %v", err)
	}
	if _, err := c.TriggerActionWithTier("inst-1", "resize", "large"); err != nil {
		t.Errorf("TriggerActionWithTier failed: %v", err)
	}
	if _, err := c.GetCustomerApps(""); err != nil {
		t.Errorf("GetCustomerApps failed: %v", err)
	}
	if _, err := c.GetCustomerApp("inst-1"); err != nil {
		t.Errorf("GetCustomerApp failed: %v", err)
	}
	if _, err := c.GetCredentials("inst-1"); err != nil {
		t.Errorf("GetCredentials failed: %v", err)
	}
}

func TestOperationsAndBackups(t *testing.T) {
	client := newTestHTTPClient(func(r *http.Request) (*http.Response, error) {
		switch r.URL.Path {
		case "/api/v1/operations":
			if r.URL.Query().Get("id") == "op-1" {
				return jsonResponse(http.StatusOK, map[string]interface{}{"status": "succeeded"})
			}
			return jsonResponse(http.StatusOK, []map[string]interface{}{{"id": "op-1"}})
		case "/api/v1/operations/op-1/retry":
			return jsonResponse(http.StatusOK, map[string]interface{}{"id": "op-1"})
		case "/api/v1/backups":
			return jsonResponse(http.StatusOK, []map[string]interface{}{{"id": "bk-1"}})
		case "/api/v1/backups/bk-1":
			return jsonResponse(http.StatusOK, map[string]interface{}{"id": "bk-1"})
		case "/api/v1/backups/restore":
			return jsonResponse(http.StatusAccepted, admiral.RestoreBackupResponse{OperationID: "op-1"})
		case "/api/admin/instances/inst-1/inspect":
			if r.Method == "POST" {
				return jsonResponse(http.StatusAccepted, admiral.OperationResponse{OperationID: "op-1"})
			}
			return jsonResponse(http.StatusOK, map[string]interface{}{"result": "ok"})
		}
		return jsonResponse(http.StatusNotFound, nil)
	})
	c := &Client{serverURL: "https://example.com", token: "token", http: client}

	if _, err := c.GetOperations(); err != nil {
		t.Errorf("GetOperations failed: %v", err)
	}
	if _, err := c.GetOperation("op-1"); err != nil {
		t.Errorf("GetOperation failed: %v", err)
	}
	if _, err := c.WaitForOperation("op-1", 1*time.Millisecond, time.Second); err != nil {
		t.Errorf("WaitForOperation failed: %v", err)
	}
	if _, err := c.RetryOperation("op-1"); err != nil {
		t.Errorf("RetryOperation failed: %v", err)
	}
	if _, err := c.GetBackups(); err != nil {
		t.Errorf("GetBackups failed: %v", err)
	}
	if _, err := c.GetBackup("bk-1"); err != nil {
		t.Errorf("GetBackup failed: %v", err)
	}
	if _, err := c.RestoreBackup(admiral.RestoreBackupRequest{}); err != nil {
		t.Errorf("RestoreBackup failed: %v", err)
	}
	if _, err := c.TriggerInspect("inst-1"); err != nil {
		t.Errorf("TriggerInspect failed: %v", err)
	}
	if _, err := c.GetInspectResult("inst-1"); err != nil {
		t.Errorf("GetInspectResult failed: %v", err)
	}
}

func TestWaitForOperationTimesOut(t *testing.T) {
	httpClient := newTestHTTPClient(func(r *http.Request) (*http.Response, error) {
		return jsonResponse(http.StatusOK, map[string]interface{}{"status": "running"})
	})
	c := &Client{serverURL: "https://example.com", token: "token", http: httpClient}

	if _, err := c.WaitForOperation("op-timeout", time.Millisecond, 5*time.Millisecond); err == nil || !strings.Contains(err.Error(), "timed out") {
		t.Fatalf("expected timeout error, got %v", err)
	}
}

func TestUsersManagement(t *testing.T) {
	client := newTestHTTPClient(func(r *http.Request) (*http.Response, error) {
		switch r.URL.Path {
		case "/api/admin/users":
			if r.Method == "POST" {
				return jsonResponse(http.StatusCreated, map[string]interface{}{"username": "user1"})
			}
			return jsonResponse(http.StatusOK, []map[string]interface{}{{"username": "user1"}})
		case "/api/admin/users/user1/set-password":
			return jsonResponse(http.StatusOK, nil)
		}
		return jsonResponse(http.StatusNotFound, nil)
	})
	c := &Client{serverURL: "https://example.com", token: "token", http: client}

	if _, err := c.CreateUser("user1", "pass", "admin"); err != nil {
		t.Errorf("CreateUser failed: %v", err)
	}
	if _, err := c.ListUsers(); err != nil {
		t.Errorf("ListUsers failed: %v", err)
	}
	if err := c.SetPassword("user1", "new-pass"); err != nil {
		t.Errorf("SetPassword failed: %v", err)
	}
}

func TestRotateSecrets(t *testing.T) {
	client := newTestHTTPClient(func(r *http.Request) (*http.Response, error) {
		if r.Method != http.MethodPost || r.URL.Path != "/api/v1/secrets/rotate" {
			return jsonResponse(http.StatusNotFound, nil)
		}
		return jsonResponse(http.StatusOK, map[string]int{"rotated_count": 5})
	})

	c := &Client{serverURL: "https://example.com", token: "token", http: client}
	res, err := c.RotateSecrets()
	if err != nil {
		t.Fatalf("RotateSecrets failed: %v", err)
	}
	if res["rotated_count"] != 5 {
		t.Fatalf("unexpected rotated count: %v", res)
	}
}

func TestWithHTTPClientOption(t *testing.T) {
	customClient := &http.Client{}
	c := &Client{http: &http.Client{}}
	WithHTTPClient(customClient)(c)
	if c.http != customClient {
		t.Fatal("WithHTTPClient option failed to set http client")
	}
}

func TestNewClientValidations(t *testing.T) {
	// invalid URL scheme
	_, err := New("http://invalid.com", "token", "")
	if err == nil {
		t.Fatal("expected error with http scheme")
	}

	// valid URL scheme
	c, err := New("https://valid.com", "token", "")
	if err != nil {
		t.Fatalf("New failed: %v", err)
	}
	if c.serverURL != "https://valid.com" {
		t.Fatalf("unexpected serverURL: %s", c.serverURL)
	}
}

type mockNetError struct {
	error
}

func (e mockNetError) Timeout() bool   { return true }
func (e mockNetError) Temporary() bool { return true }

var _ net.Error = mockNetError{}

func TestIsRetryableNetworkError(t *testing.T) {
	if isRetryableNetworkError(nil) {
		t.Fatal("nil error should not be retryable")
	}

	connRefused := errors.New("connection refused")
	if !isRetryableNetworkError(connRefused) {
		t.Fatal("connection refused should be retryable")
	}

	noSuchHost := errors.New("no such host")
	if !isRetryableNetworkError(noSuchHost) {
		t.Fatal("no such host should be retryable")
	}

	tlsHandshake := errors.New("TLS handshake")
	if !isRetryableNetworkError(tlsHandshake) {
		t.Fatal("TLS handshake should be retryable")
	}

	genericErr := errors.New("generic error")
	if isRetryableNetworkError(genericErr) {
		t.Fatal("generic error should not be retryable")
	}

	urlErr := &url.Error{
		Op:  "Get",
		URL: "https://example.com",
		Err: errors.New("connection refused"),
	}
	if !isRetryableNetworkError(urlErr) {
		t.Fatal("url error wrapping retryable error should be retryable")
	}

	netErr := mockNetError{error: errors.New("timeout")}
	if !isRetryableNetworkError(netErr) {
		t.Fatal("timeout net.Error should be retryable")
	}

	urlNetErr := &url.Error{
		Op:  "Get",
		URL: "https://example.com",
		Err: mockNetError{error: errors.New("timeout")},
	}
	if !isRetryableNetworkError(urlNetErr) {
		t.Fatal("url error wrapping timeout net.Error should be retryable")
	}
}
