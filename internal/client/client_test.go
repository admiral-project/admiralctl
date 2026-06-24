// SPDX-FileCopyrightText: William Moreno Reyes CP | MBA
// SPDX-License-Identifier: Apache-2.0

package client

import (
	"encoding/json"
	"errors"
	"io"
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

func TestNodeManagement(t *testing.T) {
	client := newTestHTTPClient(func(r *http.Request) (*http.Response, error) {
		switch r.URL.Path {
		case "/api/v1/nodes":
			if r.Method == http.MethodPost {
				return jsonResponse(http.StatusOK, nil)
			}
			return jsonResponse(http.StatusOK, []map[string]interface{}{{"id": "node1"}})
		case "/api/v1/nodes/node1":
			if r.Method == http.MethodGet {
				return jsonResponse(http.StatusOK, map[string]interface{}{"id": "node1"})
			}
			return jsonResponse(http.StatusOK, nil)
		case "/api/v1/nodes/node1/enable", "/api/v1/nodes/node1/disable":
			return jsonResponse(http.StatusOK, nil)
		case "/api/v1/nodes/node1/ready":
			return jsonResponse(http.StatusOK, map[string]interface{}{"ready": true})
		}
		return jsonResponse(http.StatusNotFound, nil)
	})

	c := &Client{serverURL: "https://example.com", token: "token", http: client}

	if err := c.RegisterNode(admiral.RegisterNodeRequest{NodeID: "node1"}); err != nil {
		t.Errorf("RegisterNode: %v", err)
	}
	if _, err := c.GetNodes(); err != nil {
		t.Errorf("GetNodes: %v", err)
	}
	if _, err := c.GetNode("node1"); err != nil {
		t.Errorf("GetNode: %v", err)
	}
	if err := c.EnableNode("node1"); err != nil {
		t.Errorf("EnableNode: %v", err)
	}
	if err := c.DisableNode("node1"); err != nil {
		t.Errorf("DisableNode: %v", err)
	}
	if _, err := c.NodeReady("node1"); err != nil {
		t.Errorf("NodeReady: %v", err)
	}
	if err := c.RemoveNode("node1"); err != nil {
		t.Errorf("RemoveNode: %v", err)
	}
}

func TestAppAndInstanceManagement(t *testing.T) {
	client := newTestHTTPClient(func(r *http.Request) (*http.Response, error) {
		if r.URL.Path == "/api/v1/apps" {
			if r.Method == http.MethodPost {
				return jsonResponse(http.StatusOK, map[string]interface{}{"name": "myapp"})
			}
			return jsonResponse(http.StatusOK, []map[string]interface{}{{"name": "myapp"}})
		}
		if r.URL.Path == "/api/v1/apps/myapp" {
			return jsonResponse(http.StatusOK, map[string]interface{}{"name": "myapp"})
		}
		if r.URL.Path == "/api/v1/apps/myapp/status" {
			return jsonResponse(http.StatusOK, nil)
		}
		if r.URL.Path == "/api/v1/customer-apps" {
			if r.Method == http.MethodPost {
				return jsonResponse(http.StatusAccepted, admiral.ProvisionResponse{OperationID: "op1"})
			}
			return jsonResponse(http.StatusOK, []map[string]interface{}{{"id": "inst1"}})
		}
		if r.URL.Path == "/api/v1/customer-apps/action" {
			return jsonResponse(http.StatusAccepted, admiral.OperationResponse{OperationID: "op1"})
		}
		if r.URL.Path == "/api/v1/customer-apps/inst1" {
			return jsonResponse(http.StatusOK, map[string]interface{}{"id": "inst1"})
		}
		if r.URL.Path == "/api/admin/instances/inst1/inspect" {
			if r.Method == http.MethodPost {
				return jsonResponse(http.StatusAccepted, admiral.OperationResponse{OperationID: "op1"})
			}
			return jsonResponse(http.StatusOK, map[string]interface{}{"result": "ok"})
		}
		if r.URL.Path == "/api/admin/instances/inst1/migrate" {
			return jsonResponse(http.StatusAccepted, admiral.MigrateAppResponse{OperationID: "op1"})
		}
		return jsonResponse(http.StatusNotFound, nil)
	})

	c := &Client{serverURL: "https://example.com", token: "token", http: client}

	if _, err := c.ApplyApp("yaml"); err != nil {
		t.Errorf("ApplyApp: %v", err)
	}
	if _, err := c.GetApps(); err != nil {
		t.Errorf("GetApps: %v", err)
	}
	if _, err := c.GetApp("myapp"); err != nil {
		t.Errorf("GetApp: %v", err)
	}
	if err := c.UpdateAppStatus("myapp", "active"); err != nil {
		t.Errorf("UpdateAppStatus: %v", err)
	}
	if _, err := c.ProvisionApp(admiral.ProvisionRequest{}); err != nil {
		t.Errorf("ProvisionApp: %v", err)
	}
	if _, err := c.TriggerAction("inst1", "restart"); err != nil {
		t.Errorf("TriggerAction: %v", err)
	}
	if _, err := c.TriggerActionWithService("inst1", "backup", "db"); err != nil {
		t.Errorf("TriggerActionWithService: %v", err)
	}
	if _, err := c.TriggerActionWithTier("inst1", "resize", "large"); err != nil {
		t.Errorf("TriggerActionWithTier: %v", err)
	}
	if _, err := c.GetCustomerApps(); err != nil {
		t.Errorf("GetCustomerApps: %v", err)
	}
	if _, err := c.GetCustomerApp("inst1"); err != nil {
		t.Errorf("GetCustomerApp: %v", err)
	}
	if _, err := c.TriggerInspect("inst1"); err != nil {
		t.Errorf("TriggerInspect: %v", err)
	}
	if _, err := c.GetInspectResult("inst1"); err != nil {
		t.Errorf("GetInspectResult: %v", err)
	}
	if _, err := c.MigrateInstance("inst1", "node2"); err != nil {
		t.Errorf("MigrateInstance: %v", err)
	}
}

func TestOperationAndBackupManagement(t *testing.T) {
	client := newTestHTTPClient(func(r *http.Request) (*http.Response, error) {
		if r.URL.Path == "/api/v1/operations" {
			return jsonResponse(http.StatusOK, []map[string]interface{}{{"status": "succeeded"}})
		}
		if strings.HasPrefix(r.URL.Path, "/api/v1/operations/") && strings.HasSuffix(r.URL.Path, "/retry") {
			return jsonResponse(http.StatusOK, map[string]interface{}{"id": "op1"})
		}
		if r.URL.Path == "/api/v1/backups/restore" {
			return jsonResponse(http.StatusAccepted, admiral.RestoreBackupResponse{OperationID: "op1"})
		}
		if r.URL.Path == "/api/v1/backups" {
			return jsonResponse(http.StatusOK, []map[string]interface{}{{"id": "bk1"}})
		}
		if r.URL.Path == "/api/v1/backups/bk1" {
			return jsonResponse(http.StatusOK, map[string]interface{}{"id": "bk1"})
		}
		if r.URL.Path == "/api/admin/backups/bk1" {
			return jsonResponse(http.StatusOK, nil)
		}
		if r.URL.Path == "/api/admin/backups/prune" {
			return jsonResponse(http.StatusOK, nil)
		}
		return jsonResponse(http.StatusNotFound, nil)
	})

	c := &Client{serverURL: "https://example.com", token: "token", http: client}

	if _, err := c.GetOperations(); err != nil {
		t.Errorf("GetOperations: %v", err)
	}
	// GetOperation uses query param
	// WaitForOperation uses GetOperation
	if _, err := c.RetryOperation("op1"); err != nil {
		t.Errorf("RetryOperation: %v", err)
	}
	if _, err := c.RestoreBackup(admiral.RestoreBackupRequest{}); err != nil {
		t.Errorf("RestoreBackup: %v", err)
	}
	if _, err := c.GetBackups(); err != nil {
		t.Errorf("GetBackups: %v", err)
	}
	if _, err := c.GetBackup("bk1"); err != nil {
		t.Errorf("GetBackup: %v", err)
	}
	if err := c.PruneBackups(); err != nil {
		t.Errorf("PruneBackups: %v", err)
	}
}

func TestUserAndRouteManagement(t *testing.T) {
	client := newTestHTTPClient(func(r *http.Request) (*http.Response, error) {
		if r.URL.Path == "/api/admin/users" {
			if r.Method == http.MethodPost {
				return jsonResponse(http.StatusCreated, map[string]interface{}{"username": "user1"})
			}
			return jsonResponse(http.StatusOK, []map[string]interface{}{{"username": "user1"}})
		}
		if strings.HasSuffix(r.URL.Path, "/set-password") {
			return jsonResponse(http.StatusOK, nil)
		}
		return jsonResponse(http.StatusNotFound, nil)
	})

	c := &Client{serverURL: "https://example.com", token: "token", http: client}

	if _, err := c.CreateUser("user1", "pass", "admin"); err != nil {
		t.Errorf("CreateUser: %v", err)
	}
	if err := c.SetPassword("user1", "newpass"); err != nil {
		t.Errorf("SetPassword: %v", err)
	}
	if _, err := c.ListUsers(); err != nil {
		t.Errorf("ListUsers: %v", err)
	}
}

func TestClientOptionsAndNew(t *testing.T) {
	c, err := New("https://localhost:8080", "token", "",
		WithTimeout(10*time.Second),
		WithRetries(5, 500*time.Millisecond),
		WithOperator("jules"),
	)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	if c.serverURL != "https://localhost:8080" {
		t.Errorf("expected server URL, got %q", c.serverURL)
	}
	if c.maxRetries != 5 {
		t.Errorf("expected maxRetries 5, got %d", c.maxRetries)
	}
	if c.operator != "jules" {
		t.Errorf("expected operator jules, got %q", c.operator)
	}

	_, err = New("http://localhost:8080", "token", "")
	if err == nil {
		t.Error("expected error for http URL")
	}
}

func TestNetworkErrorRetryable(t *testing.T) {
	tests := []struct {
		err  error
		want bool
	}{
		{err: nil, want: false},
		{err: errors.New("generic"), want: false},
		{err: errors.New("connection refused"), want: true},
		{err: errors.New("no such host"), want: true},
		{err: errors.New("TLS handshake"), want: true},
		{err: &url.Error{Err: errors.New("connection refused")}, want: true},
	}

	for _, tt := range tests {
		if got := isRetryableNetworkError(tt.err); got != tt.want {
			t.Errorf("isRetryableNetworkError(%v) = %v, want %v", tt.err, got, tt.want)
		}
	}
}

func TestGetOperationAndWait(t *testing.T) {
	attempts := 0
	client := newTestHTTPClient(func(r *http.Request) (*http.Response, error) {
		if r.URL.Path == "/api/v1/operations" {
			attempts++
			status := "running"
			if attempts >= 2 {
				status = "succeeded"
			}
			return jsonResponse(http.StatusOK, map[string]interface{}{"status": status})
		}
		return jsonResponse(http.StatusNotFound, nil)
	})

	c := &Client{serverURL: "https://example.com", token: "token", http: client}

	op, err := c.GetOperation("op1")
	if err != nil {
		t.Fatalf("GetOperation: %v", err)
	}
	if op["status"] != "running" {
		t.Errorf("expected status running, got %v", op["status"])
	}

	op, err = c.WaitForOperation("op1", 1*time.Millisecond)
	if err != nil {
		t.Fatalf("WaitForOperation: %v", err)
	}
	if op["status"] != "succeeded" {
		t.Errorf("expected status succeeded, got %v", op["status"])
	}
}

func TestSanitizeErrorBody(t *testing.T) {
	tests := []struct {
		name string
		resp []byte
		want string
	}{
		{name: "empty", resp: []byte(""), want: ""},
		{name: "structured error", resp: []byte(`{"error":"fail"}`), want: "fail"},
		{name: "structured message", resp: []byte(`{"message":"fail"}`), want: "fail"},
		{name: "unstructured short", resp: []byte("short error"), want: "server returned an unstructured error response"},
		{name: "unstructured long", resp: []byte(strings.Repeat("a", 150)), want: "server returned an unstructured error response"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := sanitizeErrorBody(tt.resp); got != tt.want {
				t.Errorf("sanitizeErrorBody() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestGetBackupsPaged(t *testing.T) {
	client := newTestHTTPClient(func(r *http.Request) (*http.Response, error) {
		return jsonResponse(http.StatusOK, pagedBackupResponse{
			Items: []map[string]interface{}{{"id": "bk2"}},
		})
	})

	c := &Client{serverURL: "https://example.com", token: "token", http: client}
	backups, err := c.GetBackups()
	if err != nil {
		t.Fatalf("GetBackups: %v", err)
	}
	if len(backups) != 1 || backups[0]["id"] != "bk2" {
		t.Errorf("unexpected backups: %+v", backups)
	}
}
