// Copyright (c) Dutch Analytics B.V. 2026
// SPDX-License-Identifier: MPL-2.0

package client

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"mime/multipart"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const (
	maxRetries     = 3
	baseRetryDelay = 500 * time.Millisecond
)

// UbiOpsClient is a thin wrapper around net/http for calling the UbiOps API.
type UbiOpsClient struct {
	BaseURL    string
	APIToken   string
	HTTPClient *http.Client
	UserAgent  string
}

// UbiOpsError represents an error response from the UbiOps API.
type UbiOpsError struct {
	StatusCode int
	Message    string
	Detail     string
}

func (e *UbiOpsError) Error() string {
	if e.Detail != "" {
		return fmt.Sprintf("UbiOps API error (HTTP %d): %s – %s", e.StatusCode, e.Message, e.Detail)
	}
	return fmt.Sprintf("UbiOps API error (HTTP %d): %s", e.StatusCode, e.Message)
}

// NewUbiOpsClient creates a new UbiOps API client.
// resolveIP, if non-empty, overrides DNS for the base URL host (like curl --resolve).
func NewUbiOpsClient(baseURL, apiToken, providerVersion, resolveIP string) *UbiOpsClient {
	// Strip a redundant leading "Token " - UbiOps's dashboard hands tokens out
	// pre-fixed for copy-paste.
	apiToken = strings.TrimPrefix(strings.TrimPrefix(apiToken, "Token "), "token ")
	httpClient := &http.Client{Timeout: 60 * time.Second}
	// Escape hatch for self-hosted UbiOps behind an internal CA not in the system
	// trust store; off by default, opt-in only.
	insecure := os.Getenv("UBIOPS_TF_INSECURE_SKIP_VERIFY") == "1"
	if resolveIP != "" || insecure {
		transport := &http.Transport{
			TLSClientConfig:     &tls.Config{InsecureSkipVerify: insecure}, //nolint:gosec // opt-in via UBIOPS_TF_INSECURE_SKIP_VERIFY only
			TLSHandshakeTimeout: 10 * time.Second,
		}
		if resolveIP != "" {
			transport.DialContext = func(ctx context.Context, network, addr string) (net.Conn, error) {
				host, port, err := net.SplitHostPort(addr)
				if err != nil {
					return nil, err
				}
				_ = host // replaced by resolveIP
				return (&net.Dialer{Timeout: 30 * time.Second}).DialContext(ctx, network, net.JoinHostPort(resolveIP, port))
			}
		}
		httpClient.Transport = transport
	}
	return &UbiOpsClient{
		BaseURL:    baseURL,
		APIToken:   apiToken,
		HTTPClient: httpClient,
		UserAgent:  fmt.Sprintf("terraform-provider-ubiops/%s", providerVersion),
	}
}

// Get performs a GET request and decodes the JSON response into result.
func (c *UbiOpsClient) Get(ctx context.Context, path string, result any) error {
	return c.Do(ctx, http.MethodGet, path, nil, result)
}

// Post performs a POST request with a JSON body and decodes the response into result.
func (c *UbiOpsClient) Post(ctx context.Context, path string, body any, result any) error {
	return c.Do(ctx, http.MethodPost, path, body, result)
}

// Patch performs a PATCH request with a JSON body and decodes the response into result.
func (c *UbiOpsClient) Patch(ctx context.Context, path string, body any, result any) error {
	return c.Do(ctx, http.MethodPatch, path, body, result)
}

// Delete performs a DELETE request.
func (c *UbiOpsClient) Delete(ctx context.Context, path string) error {
	return c.Do(ctx, http.MethodDelete, path, nil, nil)
}

// Upload performs a multipart file upload to the given path.
func (c *UbiOpsClient) Upload(ctx context.Context, path string, filePath string, result any) error {
	f, err := os.Open(filePath)
	if err != nil {
		return fmt.Errorf("failed to open file %s: %w", filePath, err)
	}
	defer f.Close()

	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)

	part, err := writer.CreateFormFile("file", filepath.Base(filePath))
	if err != nil {
		return fmt.Errorf("failed to create form file: %w", err)
	}

	if _, err := io.Copy(part, f); err != nil {
		return fmt.Errorf("failed to copy file contents: %w", err)
	}

	if err := writer.Close(); err != nil {
		return fmt.Errorf("failed to close multipart writer: %w", err)
	}

	return c.doUpload(ctx, path, &buf, writer.FormDataContentType(), result)
}

// doUpload performs an HTTP POST with a pre-built body and content type.
func (c *UbiOpsClient) doUpload(ctx context.Context, path string, body *bytes.Buffer, contentType string, result any) error {
	return c.execute(ctx, http.MethodPost, path, body.Bytes(), contentType, result)
}

// Do performs an HTTP request with JSON serialization, authentication, and retry logic.
func (c *UbiOpsClient) Do(ctx context.Context, method, path string, body any, result any) error {
	var bodyBytes []byte
	var contentType string
	if body != nil {
		var err error
		bodyBytes, err = json.Marshal(body)
		if err != nil {
			return fmt.Errorf("failed to marshal request body: %w", err)
		}
		contentType = "application/json"
	}

	return c.execute(ctx, method, path, bodyBytes, contentType, result)
}

// execute sends an HTTP request with retry/backoff, authentication headers, and JSON
// decoding into result. contentType is set on the request only when non-empty.
func (c *UbiOpsClient) execute(ctx context.Context, method, path string, bodyBytes []byte, contentType string, result any) error {
	url := c.BaseURL + path

	var lastErr error
	for attempt := range maxRetries {
		var bodyReader io.Reader
		if bodyBytes != nil {
			bodyReader = bytes.NewReader(bodyBytes)
		}

		req, err := http.NewRequestWithContext(ctx, method, url, bodyReader)
		if err != nil {
			return fmt.Errorf("failed to create request: %w", err)
		}

		req.Header.Set("Authorization", fmt.Sprintf("Token %s", c.APIToken))
		req.Header.Set("User-Agent", c.UserAgent)
		if contentType != "" {
			req.Header.Set("Content-Type", contentType)
		}
		req.Header.Set("Accept", "application/json")

		resp, err := c.HTTPClient.Do(req)
		if err != nil {
			lastErr = fmt.Errorf("request failed: %w", err)
			// Always retry on transport-level errors (DNS, connection refused, timeout).
			backoff(attempt)
			continue
		}

		if isRetryable(resp.StatusCode) {
			lastErr = parseError(resp)
			resp.Body.Close()
			backoff(attempt)
			continue
		}

		if resp.StatusCode >= 400 {
			err := parseError(resp)
			resp.Body.Close()
			return err
		}

		// No content (e.g. 204 on DELETE).
		if resp.StatusCode == http.StatusNoContent || result == nil {
			resp.Body.Close()
			return nil
		}

		err = json.NewDecoder(resp.Body).Decode(result)
		resp.Body.Close()
		if err != nil {
			return fmt.Errorf("failed to decode response: %w", err)
		}

		return nil
	}

	return lastErr
}

// IsNotFound returns true if the error is a 404 response from the UbiOps API.
func IsNotFound(err error) bool {
	if apiErr, ok := err.(*UbiOpsError); ok {
		return apiErr.StatusCode == http.StatusNotFound
	}
	return false
}

// isRetryable returns true for status codes that should be retried.
func isRetryable(statusCode int) bool {
	return statusCode == http.StatusTooManyRequests ||
		statusCode >= http.StatusInternalServerError
}

// backoff sleeps for an exponentially increasing duration.
func backoff(attempt int) {
	delay := baseRetryDelay * time.Duration(math.Pow(2, float64(attempt)))
	time.Sleep(delay)
}

// parseError reads the response body and returns a structured UbiOpsError.
func parseError(resp *http.Response) error {
	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return &UbiOpsError{
			StatusCode: resp.StatusCode,
			Message:    fmt.Sprintf("HTTP %d (failed to read response body)", resp.StatusCode),
		}
	}

	// Try to parse as JSON error response.
	var errResp struct {
		Error  string `json:"error"`
		Detail string `json:"detail"`
	}
	if json.Unmarshal(bodyBytes, &errResp) == nil && (errResp.Error != "" || errResp.Detail != "") {
		msg := errResp.Error
		if msg == "" {
			msg = http.StatusText(resp.StatusCode)
		}
		return &UbiOpsError{
			StatusCode: resp.StatusCode,
			Message:    msg,
			Detail:     errResp.Detail,
		}
	}

	return &UbiOpsError{
		StatusCode: resp.StatusCode,
		Message:    string(bodyBytes),
	}
}
