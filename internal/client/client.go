// SPDX-FileCopyrightText: William Moreno Reyes CP | MBA
// SPDX-License-Identifier: Apache-2.0

package client

import (
	"bytes"
	"crypto/rand"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/admiral-project/admiral/admirald/pkg/admiral"
	"github.com/admiral-project/admiral/admirald/pkg/admiral/tlsconfig"
)

type Client struct {
	serverURL  string
	token      string
	operator   string
	http       *http.Client
	maxRetries int
	retryDelay time.Duration
}

type ClientOption func(*Client)

type ProvisionRejectedError struct {
	Response admiral.ProvisioningRejectedResponse
}

func (e *ProvisionRejectedError) Error() string {
	message := strings.TrimSpace(e.Response.Message)
	if message == "" {
		message = strings.TrimSpace(e.Response.Error)
	}
	if message == "" {
		message = "provisioning rejected by policy"
	}
	if e.Response.OperationID != "" {
		return fmt.Sprintf("%s (operation_id=%s)", message, e.Response.OperationID)
	}
	return message
}

func parsePolicyRejectedError(status int, resp []byte) error {
	if status != http.StatusServiceUnavailable {
		return nil
	}
	var rejected admiral.ProvisioningRejectedResponse
	if err := json.Unmarshal(resp, &rejected); err != nil {
		return nil
	}
	if rejected.Code == "" && rejected.OperationID == "" {
		return nil
	}
	return &ProvisionRejectedError{Response: rejected}
}

func WithTimeout(d time.Duration) ClientOption {
	return func(c *Client) {
		c.http.Timeout = d
	}
}

func WithRetries(maxRetries int, baseDelay time.Duration) ClientOption {
	return func(c *Client) {
		c.maxRetries = maxRetries
		c.retryDelay = baseDelay
	}
}

func WithOperator(operator string) ClientOption {
	return func(c *Client) {
		c.operator = operator
	}
}

func New(serverURL, token, caCertFile string, opts ...ClientOption) (*Client, error) {
	if err := tlsconfig.ValidateURLScheme(serverURL, "https"); err != nil {
		return nil, err
	}
	clientTLSConfig, err := tlsconfig.NewClientConfig(caCertFile)
	if err != nil {
		return nil, err
	}

	c := &Client{
		serverURL: serverURL,
		token:     token,
		http: &http.Client{
			Timeout: 30 * time.Second,
			Transport: &http.Transport{
				TLSClientConfig: clientTLSConfig,
			},
		},
		maxRetries: 3,
		retryDelay: 1 * time.Second,
	}
	for _, opt := range opts {
		opt(c)
	}
	return c, nil
}

func (c *Client) request(method, path string, body []byte) ([]byte, int, error) {
	u := fmt.Sprintf("%s%s", c.serverURL, path)
	var lastErr error
	var lastCode int

	for attempt := 0; attempt <= c.maxRetries; attempt++ {
		if attempt > 0 {
			time.Sleep(c.backoff(attempt))
		}

		respBytes, statusCode, err := c.doRequest(method, u, body)
		if err != nil {
			lastErr = err
			continue
		}

		if statusCode >= 500 {
			lastErr = fmt.Errorf("server error: HTTP %d", statusCode)
			lastCode = statusCode
			continue
		}

		return respBytes, statusCode, nil
	}

	if lastErr != nil {
		return nil, lastCode, fmt.Errorf("request failed after %d retries: %w", c.maxRetries, lastErr)
	}
	return nil, lastCode, fmt.Errorf("request failed after %d retries (HTTP %d)", c.maxRetries, lastCode)
}

func (c *Client) doRequest(method, url string, body []byte) ([]byte, int, error) {
	req, err := http.NewRequest(method, url, bytes.NewBuffer(body))
	if err != nil {
		return nil, 0, err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Admiral-Token", c.token)
	if c.operator != "" {
		req.Header.Set("X-Admiral-Operator", c.operator)
	}

	resp, err := c.http.Do(req)
	if err != nil {
		if isRetryableNetworkError(err) {
			return nil, 0, fmt.Errorf("network: %w", err)
		}
		return nil, 0, err
	}
	defer resp.Body.Close()

	respBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, resp.StatusCode, err
	}

	return respBytes, resp.StatusCode, nil
}

func isRetryableNetworkError(err error) bool {
	if err == nil {
		return false
	}
	if uerr, ok := err.(*url.Error); ok {
		err = uerr.Err
	}
	if nerr, ok := err.(net.Error); ok {
		return nerr.Timeout() || nerr.Temporary()
	}
	if strings.Contains(err.Error(), "connection refused") {
		return true
	}
	if strings.Contains(err.Error(), "no such host") {
		return true
	}
	if strings.Contains(err.Error(), "TLS handshake") {
		return true
	}
	return false
}

func (c *Client) backoff(attempt int) time.Duration {
	n := 1 << uint(attempt-1)
	base := c.retryDelay * time.Duration(n)
	jitter := time.Duration(cryptoRandFloat64() * float64(base) * 0.1)
	return base + jitter
}

func cryptoRandFloat64() float64 {
	var b [8]byte
	if _, err := rand.Read(b[:]); err != nil {
		return 0.5
	}
	return float64(binary.BigEndian.Uint64(b[:])) / float64(math.MaxUint64)
}

func formatHTTPError(operation string, status int, resp []byte) error {
	message := fmt.Sprintf("%s failed: HTTP %d", operation, status)
	if detail := sanitizeErrorBody(resp); detail != "" {
		message += " - " + detail
	}
	return fmt.Errorf("%s", message)
}

func sanitizeErrorBody(resp []byte) string {
	trimmed := strings.TrimSpace(string(resp))
	if trimmed == "" {
		return ""
	}

	var structured map[string]interface{}
	if err := json.Unmarshal(resp, &structured); err == nil {
		if raw, ok := structured["error"].(string); ok {
			return strings.TrimSpace(raw)
		}
		if raw, ok := structured["message"].(string); ok {
			return strings.TrimSpace(raw)
		}
		return ""
	}

	if len(trimmed) > 120 {
		trimmed = trimmed[:120]
	}
	return "server returned an unstructured error response"
}

func (c *Client) GetStatus() (map[string]interface{}, error) {
	resp, status, err := c.request("GET", "/health", nil)
	if err != nil {
		return nil, err
	}
	if status != http.StatusOK {
		return nil, fmt.Errorf("status check failed: HTTP %d", status)
	}

	var res map[string]interface{}
	if err := json.Unmarshal(resp, &res); err != nil {
		return nil, err
	}
	return res, nil
}

func (c *Client) RegisterNode(node admiral.RegisterNodeRequest) error {
	body, _ := json.Marshal(node)
	resp, status, err := c.request("POST", "/api/v1/nodes", body)
	if err != nil {
		return err
	}
	if status != http.StatusOK {
		return formatHTTPError("register node", status, resp)
	}
	return nil
}

func (c *Client) GetNodes() ([]map[string]interface{}, error) {
	resp, status, err := c.request("GET", "/api/v1/nodes", nil)
	if err != nil {
		return nil, err
	}
	if status != http.StatusOK {
		return nil, formatHTTPError("fetch nodes", status, resp)
	}

	var list []map[string]interface{}
	if err := json.Unmarshal(resp, &list); err != nil {
		return nil, err
	}
	return list, nil
}

func (c *Client) GetNode(id string) (map[string]interface{}, error) {
	resp, status, err := c.request("GET", "/api/v1/nodes/"+url.PathEscape(id), nil)
	if err != nil {
		return nil, err
	}
	if status != http.StatusOK {
		return nil, formatHTTPError("fetch node", status, resp)
	}

	var node map[string]interface{}
	if err := json.Unmarshal(resp, &node); err != nil {
		return nil, err
	}
	return node, nil
}

func (c *Client) EnableNode(id string) error {
	resp, status, err := c.request("POST", "/api/v1/nodes/"+url.PathEscape(id)+"/enable", nil)
	if err != nil {
		return err
	}
	if status != http.StatusOK {
		return formatHTTPError("enable node", status, resp)
	}
	return nil
}

func (c *Client) DisableNode(id string) error {
	resp, status, err := c.request("POST", "/api/v1/nodes/"+url.PathEscape(id)+"/disable", nil)
	if err != nil {
		return err
	}
	if status != http.StatusOK {
		return formatHTTPError("disable node", status, resp)
	}
	return nil
}

func (c *Client) ApplyApp(yamlContent string) (string, error) {
	body, _ := json.Marshal(map[string]string{"yaml": yamlContent})
	resp, status, err := c.request("POST", "/api/v1/apps", body)
	if err != nil {
		return "", err
	}
	if status != http.StatusOK {
		return "", formatHTTPError("apply app definition", status, resp)
	}

	var res map[string]interface{}
	if err := json.Unmarshal(resp, &res); err != nil {
		return "", fmt.Errorf("parse response: %w", err)
	}
	if name, ok := res["name"].(string); ok {
		return name, nil
	}
	return "applied", nil
}

func (c *Client) GetApps() ([]map[string]interface{}, error) {
	resp, status, err := c.request("GET", "/api/v1/apps", nil)
	if err != nil {
		return nil, err
	}
	if status != http.StatusOK {
		return nil, formatHTTPError("fetch app definitions", status, resp)
	}

	var list []map[string]interface{}
	if err := json.Unmarshal(resp, &list); err != nil {
		return nil, err
	}
	return list, nil
}

func (c *Client) GetApp(name string) (map[string]interface{}, error) {
	resp, status, err := c.request("GET", "/api/v1/apps/"+url.PathEscape(name), nil)
	if err != nil {
		return nil, err
	}
	if status != http.StatusOK {
		return nil, formatHTTPError("fetch app definition", status, resp)
	}

	var app map[string]interface{}
	if err := json.Unmarshal(resp, &app); err != nil {
		return nil, err
	}
	return app, nil
}

func (c *Client) UpdateAppStatus(name, status string) error {
	body, _ := json.Marshal(map[string]string{"status": status})
	resp, code, err := c.request("PATCH", fmt.Sprintf("/api/v1/apps/%s/status", url.PathEscape(name)), body)
	if err != nil {
		return err
	}
	if code != http.StatusOK {
		return formatHTTPError("update app status", code, resp)
	}
	return nil
}

func (c *Client) ProvisionApp(req admiral.ProvisionRequest) (*admiral.ProvisionResponse, error) {
	body, _ := json.Marshal(req)
	resp, status, err := c.request("POST", "/api/v1/customer-apps", body)
	if err != nil {
		return nil, err
	}
	if rejectedErr := parsePolicyRejectedError(status, resp); rejectedErr != nil {
		return nil, rejectedErr
	}
	if status != http.StatusAccepted {
		return nil, formatHTTPError("provision app", status, resp)
	}

	var res admiral.ProvisionResponse
	if err := json.Unmarshal(resp, &res); err != nil {
		return nil, err
	}
	return &res, nil
}

func (c *Client) TriggerAction(instanceID, action string) (string, error) {
	return c.TriggerActionWithService(instanceID, action, "")
}

func (c *Client) TriggerActionWithService(instanceID, action, service string) (string, error) {
	body, _ := json.Marshal(map[string]string{
		"instance_id": instanceID,
		"action":      action,
		"service":     service,
	})
	resp, status, err := c.request("POST", "/api/v1/customer-apps/action", body)
	if err != nil {
		return "", err
	}
	if status != http.StatusAccepted {
		return "", formatHTTPError(fmt.Sprintf("execute action %q", action), status, resp)
	}

	var res admiral.OperationResponse
	if err := json.Unmarshal(resp, &res); err != nil {
		return "", err
	}
	return res.OperationID, nil
}

func (c *Client) TriggerActionWithTier(instanceID, action, tier string) (string, error) {
	body, _ := json.Marshal(map[string]string{
		"instance_id": instanceID,
		"action":      action,
		"tier":        tier,
	})
	resp, status, err := c.request("POST", "/api/v1/customer-apps/action", body)
	if err != nil {
		return "", err
	}
	if rejectedErr := parsePolicyRejectedError(status, resp); rejectedErr != nil {
		return "", rejectedErr
	}
	if status != http.StatusAccepted {
		return "", formatHTTPError(fmt.Sprintf("execute action %q", action), status, resp)
	}

	var res admiral.OperationResponse
	if err := json.Unmarshal(resp, &res); err != nil {
		return "", err
	}
	return res.OperationID, nil
}

func (c *Client) GetCustomerApps() ([]map[string]interface{}, error) {
	resp, status, err := c.request("GET", "/api/v1/customer-apps", nil)
	if err != nil {
		return nil, err
	}
	if status != http.StatusOK {
		return nil, formatHTTPError("fetch customer apps", status, resp)
	}

	var list []map[string]interface{}
	if err := json.Unmarshal(resp, &list); err != nil {
		return nil, err
	}
	return list, nil
}

func (c *Client) GetOperations() ([]map[string]interface{}, error) {
	resp, status, err := c.request("GET", "/api/v1/operations", nil)
	if err != nil {
		return nil, err
	}
	if status != http.StatusOK {
		return nil, formatHTTPError("fetch operations", status, resp)
	}

	var list []map[string]interface{}
	if err := json.Unmarshal(resp, &list); err != nil {
		return nil, err
	}
	return list, nil
}

func (c *Client) GetOperation(opID string) (map[string]interface{}, error) {
	resp, status, err := c.request("GET", "/api/v1/operations?id="+url.QueryEscape(opID), nil)
	if err != nil {
		return nil, err
	}
	if status != http.StatusOK {
		return nil, formatHTTPError("fetch operation details", status, resp)
	}

	var res map[string]interface{}
	if err := json.Unmarshal(resp, &res); err != nil {
		return nil, err
	}
	return res, nil
}

func (c *Client) WaitForOperation(opID string, interval time.Duration) (map[string]interface{}, error) {
	for {
		op, err := c.GetOperation(opID)
		if err != nil {
			return nil, err
		}
		status, _ := op["status"].(string)
		switch status {
		case "succeeded", "failed", "cancelled":
			return op, nil
		}
		time.Sleep(interval)
	}
}

func (c *Client) RetryOperation(opID string) (map[string]interface{}, error) {
	resp, status, err := c.request("POST", "/api/v1/operations/"+url.PathEscape(opID)+"/retry", nil)
	if err != nil {
		return nil, err
	}
	if status != http.StatusOK {
		return nil, formatHTTPError("retry operation", status, resp)
	}
	var res map[string]interface{}
	if err := json.Unmarshal(resp, &res); err != nil {
		return nil, err
	}
	return res, nil
}

func (c *Client) GetCustomerApp(instanceID string) (map[string]interface{}, error) {
	resp, status, err := c.request("GET", "/api/v1/customer-apps/"+url.PathEscape(instanceID), nil)
	if err != nil {
		return nil, err
	}
	if status != http.StatusOK {
		return nil, formatHTTPError("fetch instance", status, resp)
	}
	var item map[string]interface{}
	if err := json.Unmarshal(resp, &item); err != nil {
		return nil, err
	}
	return item, nil
}

func (c *Client) RestoreBackup(req admiral.RestoreBackupRequest) (*admiral.RestoreBackupResponse, error) {
	body, _ := json.Marshal(req)
	resp, status, err := c.request("POST", "/api/v1/backups/restore", body)
	if err != nil {
		return nil, err
	}
	if status != http.StatusAccepted {
		return nil, formatHTTPError("restore backup", status, resp)
	}
	var res admiral.RestoreBackupResponse
	if err := json.Unmarshal(resp, &res); err != nil {
		return nil, err
	}
	return &res, nil
}

type pagedBackupResponse struct {
	Items    []map[string]interface{} `json:"items"`
	Page     int                      `json:"page"`
	PageSize int                      `json:"page_size"`
	Total    int                      `json:"total"`
}

func (c *Client) GetBackups() ([]map[string]interface{}, error) {
	resp, status, err := c.request("GET", "/api/v1/backups", nil)
	if err != nil {
		return nil, err
	}
	if status != http.StatusOK {
		return nil, formatHTTPError("fetch backups", status, resp)
	}
	var list []map[string]interface{}
	if err := json.Unmarshal(resp, &list); err == nil {
		return list, nil
	}
	var paged pagedBackupResponse
	if err := json.Unmarshal(resp, &paged); err != nil {
		return nil, fmt.Errorf("unmarshal backups response: %w", err)
	}
	return paged.Items, nil
}

func (c *Client) GetBackup(backupID string) (map[string]interface{}, error) {
	resp, status, err := c.request("GET", "/api/v1/backups/"+url.PathEscape(backupID), nil)
	if err != nil {
		return nil, err
	}
	if status != http.StatusOK {
		return nil, formatHTTPError("fetch backup", status, resp)
	}
	var item map[string]interface{}
	if err := json.Unmarshal(resp, &item); err != nil {
		return nil, err
	}
	return item, nil
}

func (c *Client) GetRoutes() ([]map[string]interface{}, error) {
	resp, status, err := c.request("GET", "/api/v1/routes", nil)
	if err != nil {
		return nil, err
	}
	if status != http.StatusOK {
		return nil, formatHTTPError("fetch routes", status, resp)
	}
	var list []map[string]interface{}
	if err := json.Unmarshal(resp, &list); err != nil {
		return nil, err
	}
	return list, nil
}

func (c *Client) GetRoute(hostname string) (map[string]interface{}, error) {
	resp, status, err := c.request("GET", "/api/v1/routes/"+url.PathEscape(hostname), nil)
	if err != nil {
		return nil, err
	}
	if status != http.StatusOK {
		return nil, formatHTTPError("fetch route", status, resp)
	}
	var item map[string]interface{}
	if err := json.Unmarshal(resp, &item); err != nil {
		return nil, err
	}
	return item, nil
}

func (c *Client) SyncRoutes() error {
	resp, status, err := c.request("POST", "/api/v1/routes", nil)
	if err != nil {
		return err
	}
	if status != http.StatusOK {
		return formatHTTPError("sync routes", status, resp)
	}
	return nil
}

func (c *Client) EnableRoute(hostname string) error {
	resp, status, err := c.request("POST", "/api/v1/routes/"+url.PathEscape(hostname)+"/enable", nil)
	if err != nil {
		return err
	}
	if status != http.StatusOK {
		return formatHTTPError("enable route", status, resp)
	}
	return nil
}

func (c *Client) DisableRoute(hostname string) error {
	resp, status, err := c.request("POST", "/api/v1/routes/"+url.PathEscape(hostname)+"/disable", nil)
	if err != nil {
		return err
	}
	if status != http.StatusOK {
		return formatHTTPError("disable route", status, resp)
	}
	return nil
}

// --- User Management ---

func (c *Client) CreateUser(username, password, role string) (map[string]interface{}, error) {
	body, _ := json.Marshal(map[string]string{
		"username": username,
		"password": password,
		"role":     role,
	})
	resp, status, err := c.request("POST", "/api/admin/users", body)
	if err != nil {
		return nil, err
	}
	if status != http.StatusCreated {
		return nil, formatHTTPError("create user", status, resp)
	}
	var res map[string]interface{}
	if err := json.Unmarshal(resp, &res); err != nil {
		return nil, err
	}
	return res, nil
}

func (c *Client) SetPassword(username, newPassword string) error {
	body, _ := json.Marshal(map[string]string{"new_password": newPassword})
	resp, status, err := c.request("POST", "/api/admin/users/"+url.PathEscape(username)+"/set-password", body)
	if err != nil {
		return err
	}
	if status != http.StatusOK {
		return formatHTTPError("set password", status, resp)
	}
	return nil
}

func (c *Client) ListUsers() ([]map[string]interface{}, error) {
	resp, status, err := c.request("GET", "/api/admin/users", nil)
	if err != nil {
		return nil, err
	}
	if status != http.StatusOK {
		return nil, formatHTTPError("list users", status, resp)
	}
	var list []map[string]interface{}
	if err := json.Unmarshal(resp, &list); err != nil {
		return nil, err
	}
	return list, nil
}
