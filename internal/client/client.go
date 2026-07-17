// SPDX-FileCopyrightText: William Moreno Reyes CP | MBA
// SPDX-License-Identifier: Apache-2.0

package client

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/binary"
	"encoding/json"
	"errors"
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

type Option func(*Client)

const maxResponseSize = 10 * 1024 * 1024

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
	if err := json.Unmarshal(resp, &rejected); err == nil {
		if rejected.Code != "" || rejected.OperationID != "" {
			return &ProvisionRejectedError{Response: rejected}
		}
	}
	return nil
}

func WithTimeout(d time.Duration) Option {
	return func(c *Client) {
		c.http.Timeout = d
	}
}

func WithRetries(maxRetries int, baseDelay time.Duration) Option {
	return func(c *Client) {
		c.maxRetries = maxRetries
		c.retryDelay = baseDelay
	}
}

func WithOperator(operator string) Option {
	return func(c *Client) {
		c.operator = operator
	}
}

// WithHTTPClient replaces the default HTTP client for an explicitly supplied
// transport, such as a test client.
func WithHTTPClient(httpClient *http.Client) Option {
	return func(c *Client) {
		if httpClient != nil {
			c.http = httpClient
		}
	}
}

func New(serverURL, token, caCertFile string, opts ...Option) (*Client, error) {
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
	req, err := http.NewRequestWithContext(context.Background(), method, url, bytes.NewBuffer(body))
	if err != nil {
		return nil, 0, err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.token)
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

	respBytes, err := io.ReadAll(io.LimitReader(resp.Body, maxResponseSize+1))
	if err != nil {
		return nil, resp.StatusCode, err
	}
	if len(respBytes) > maxResponseSize {
		return nil, resp.StatusCode, fmt.Errorf("response body exceeds %d bytes", maxResponseSize)
	}

	return respBytes, resp.StatusCode, nil
}

func isRetryableNetworkError(err error) bool {
	if err == nil {
		return false
	}
	var nerr net.Error
	var uerr *url.Error
	if errors.As(err, &uerr) {
		err = uerr.Err
	}
	if errors.As(err, &nerr) {
		return nerr.Timeout()
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
	shift := attempt - 1
	if shift < 0 {
		shift = 0
	}
	if shift > 20 {
		shift = 20
	}
	n := time.Duration(1)
	for i := 0; i < shift; i++ {
		n *= 2
	}
	base := c.retryDelay * n
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
		return "server returned an unstructured error response"
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
	body, err := json.Marshal(node)
	if err != nil {
		return fmt.Errorf("marshal node request: %w", err)
	}
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

func (c *Client) RemoveNode(id string) error {
	resp, status, err := c.request("DELETE", "/api/v1/nodes/"+url.PathEscape(id), nil)
	if err != nil {
		return err
	}
	if status == http.StatusConflict {
		var body map[string]interface{}
		if json.Unmarshal(resp, &body) == nil {
			if msg, ok := body["error"].(string); ok {
				return fmt.Errorf("%s", msg)
			}
		}
	}
	if status != http.StatusOK {
		return formatHTTPError("remove node", status, resp)
	}
	return nil
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

func (c *Client) NodeReady(id string) (map[string]interface{}, error) {
	resp, status, err := c.request("GET", "/api/v1/nodes/"+url.PathEscape(id)+"/ready", nil)
	if err != nil {
		return nil, err
	}
	if status != http.StatusOK {
		return nil, formatHTTPError("check node ready", status, resp)
	}
	var result map[string]interface{}
	if err := json.Unmarshal(resp, &result); err != nil {
		return nil, err
	}
	return result, nil
}

func (c *Client) ApplyApp(yamlContent string) (string, error) {
	body, err := json.Marshal(map[string]string{"yaml": yamlContent})
	if err != nil {
		return "", fmt.Errorf("marshal app request: %w", err)
	}
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
	body, err := json.Marshal(map[string]string{"status": status})
	if err != nil {
		return fmt.Errorf("marshal app status request: %w", err)
	}
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
	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("marshal provision request: %w", err)
	}
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
	body, err := json.Marshal(map[string]string{
		"instance_id": instanceID,
		"action":      action,
		"service":     service,
	})
	if err != nil {
		return "", fmt.Errorf("marshal action request: %w", err)
	}
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
	body, err := json.Marshal(map[string]string{
		"instance_id": instanceID,
		"action":      action,
		"tier":        tier,
	})
	if err != nil {
		return "", fmt.Errorf("marshal tier action request: %w", err)
	}
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

func (c *Client) GetCustomerApps(customerID string) ([]map[string]interface{}, error) {
	var endpoint string
	if customerID == "" {
		endpoint = "/api/v1/instances"
	} else {
		endpoint = "/api/v1/customer-apps?customer_id=" + url.QueryEscape(customerID)
	}
	resp, status, err := c.request("GET", endpoint, nil)
	if err != nil {
		return nil, err
	}
	if status != http.StatusOK {
		return nil, formatHTTPError("fetch customer apps", status, resp)
	}

	// Admin endpoint returns a paged response; customer endpoint returns a flat array.
	var list []map[string]interface{}
	if err := json.Unmarshal(resp, &list); err == nil {
		return list, nil
	}
	var paged struct {
		Items []map[string]interface{} `json:"items"`
	}
	if err := json.Unmarshal(resp, &paged); err != nil {
		return nil, fmt.Errorf("unmarshal customer apps response: %w", err)
	}
	return paged.Items, nil
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

func (c *Client) WaitForOperation(opID string, interval, timeout time.Duration) (map[string]interface{}, error) {
	if timeout <= 0 {
		return nil, fmt.Errorf("wait for operation %q: timeout must be positive", opID)
	}
	deadline := time.Now().Add(timeout)
	for {
		if time.Now().After(deadline) {
			return nil, fmt.Errorf("wait for operation %q timed out after %s", opID, timeout)
		}
		op, err := c.GetOperation(opID)
		if err != nil {
			return nil, err
		}
		status, _ := op["status"].(string)
		switch status {
		case "succeeded", "failed", "cancelled":
			return op, nil
		}
		sleepFor := interval
		if remaining := time.Until(deadline); remaining < sleepFor {
			sleepFor = remaining
		}
		if sleepFor > 0 {
			time.Sleep(sleepFor)
		}
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

func (c *Client) GetCredentials(instanceID string) ([]admiral.Credential, error) {
	resp, status, err := c.request("GET", "/api/v1/customer-apps/"+url.PathEscape(instanceID)+"/credentials", nil)
	if err != nil {
		return nil, err
	}
	if status != http.StatusOK {
		return nil, formatHTTPError("fetch credentials", status, resp)
	}
	var credentials []admiral.Credential
	if err := json.Unmarshal(resp, &credentials); err != nil {
		return nil, err
	}
	return credentials, nil
}

func (c *Client) TriggerInspect(instanceID string) (string, error) {
	resp, status, err := c.request("POST", "/api/admin/instances/"+url.PathEscape(instanceID)+"/inspect", nil)
	if err != nil {
		return "", err
	}
	if status != http.StatusAccepted {
		return "", formatHTTPError("trigger inspect", status, resp)
	}
	var res admiral.OperationResponse
	if err := json.Unmarshal(resp, &res); err != nil {
		return "", err
	}
	return res.OperationID, nil
}

func (c *Client) GetInspectResult(instanceID string) (map[string]interface{}, error) {
	resp, status, err := c.request("GET", "/api/admin/instances/"+url.PathEscape(instanceID)+"/inspect", nil)
	if err != nil {
		return nil, err
	}
	if status != http.StatusOK {
		return nil, formatHTTPError("fetch inspect result", status, resp)
	}
	var data map[string]interface{}
	if err := json.Unmarshal(resp, &data); err != nil {
		return nil, err
	}
	return data, nil
}

func (c *Client) RestoreBackup(req admiral.RestoreBackupRequest) (*admiral.RestoreBackupResponse, error) {
	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("marshal restore request: %w", err)
	}
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
	body, err := json.Marshal(map[string]string{
		"username": username,
		"password": password,
		"role":     role,
	})
	if err != nil {
		return nil, fmt.Errorf("marshal create-user request: %w", err)
	}
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
	body, err := json.Marshal(map[string]string{"new_password": newPassword})
	if err != nil {
		return fmt.Errorf("marshal set-password request: %w", err)
	}
	resp, status, err := c.request("POST", "/api/admin/users/"+url.PathEscape(username)+"/set-password", body)
	if err != nil {
		return err
	}
	if status != http.StatusOK {
		return formatHTTPError("set password", status, resp)
	}
	return nil
}

func (c *Client) RotateSecrets() (map[string]int, error) {
	body, status, err := c.request(http.MethodPost, "/api/v1/secrets/rotate", nil)
	if err != nil {
		return nil, err
	}
	if status < 200 || status >= 300 {
		return nil, fmt.Errorf("secret rotation failed: HTTP %d: %s", status, strings.TrimSpace(string(body)))
	}
	var result map[string]int
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("decode secret rotation response: %w", err)
	}
	return result, nil
}

func (c *Client) GetBackupStorageConfig() (*admiral.BackupStorageConfig, error) {
	resp, status, err := c.request("GET", "/api/admin/settings/backup-storage", nil)
	if err != nil {
		return nil, err
	}
	if status != http.StatusOK {
		return nil, formatHTTPError("get backup storage config", status, resp)
	}
	var cfg admiral.BackupStorageConfig
	if err := json.Unmarshal(resp, &cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}

func (c *Client) SetBackupStorageConfig(cfg admiral.BackupStorageConfig) error {
	body, err := json.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("marshal backup storage config: %w", err)
	}
	resp, status, err := c.request("PUT", "/api/admin/settings/backup-storage", body)
	if err != nil {
		return err
	}
	if status != http.StatusOK {
		return formatHTTPError("set backup storage config", status, resp)
	}
	return nil
}

func (c *Client) TestBackupStorageConfig() error {
	resp, status, err := c.request("POST", "/api/admin/settings/backup-storage/test", nil)
	if err != nil {
		return err
	}
	if status != http.StatusOK {
		return formatHTTPError("test backup storage config", status, resp)
	}
	return nil
}

func (c *Client) DeleteBackup(backupID string) error {
	resp, status, err := c.request("DELETE", "/api/admin/backups/"+url.PathEscape(backupID), nil)
	if err != nil {
		return err
	}
	if status != http.StatusOK && status != http.StatusAccepted {
		return formatHTTPError("delete backup", status, resp)
	}
	return nil
}

func (c *Client) PruneBackups() error {
	resp, status, err := c.request("POST", "/api/admin/backups/prune", nil)
	if err != nil {
		return err
	}
	if status != http.StatusOK {
		return formatHTTPError("prune backups", status, resp)
	}
	return nil
}

func (c *Client) MigrateInstance(instanceID, targetNodeID string) (*admiral.MigrateAppResponse, error) {
	body, err := json.Marshal(admiral.MigrateAppRequest{TargetNodeID: targetNodeID})
	if err != nil {
		return nil, fmt.Errorf("marshal migrate request: %w", err)
	}
	resp, status, err := c.request("POST", "/api/admin/instances/"+url.PathEscape(instanceID)+"/migrate", body)
	if err != nil {
		return nil, err
	}
	if status != http.StatusAccepted {
		return nil, formatHTTPError("migrate instance", status, resp)
	}
	var res admiral.MigrateAppResponse
	if err := json.Unmarshal(resp, &res); err != nil {
		return nil, err
	}
	return &res, nil
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
