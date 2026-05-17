// Copyright (c) 2026 Caputchin
// SPDX-License-Identifier: MPL-2.0

// Package client implements the HTTP transport every resource and data source
// uses to talk to the Caputchin management API.
package client

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// Client is the typed HTTP client every resource and data source uses to talk
// to the Caputchin management API. It threads the PAT through every request
// and surfaces error codes verbatim so callers can grep for the same kebab-case
// strings the routes emit.
type Client struct {
	endpoint string
	token    string
	ua       string
	http     *http.Client
}

// NewClient constructs a Client. The version string lands in the User-Agent so
// server-side telemetry can correlate provider versions to behavior.
func NewClient(endpoint, token, version string) *Client {
	return &Client{
		endpoint: strings.TrimRight(endpoint, "/"),
		token:    token,
		ua:       fmt.Sprintf("terraform-provider-caputchin/%s", version),
		http: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// APIError is the structured error returned when the API responds with a
// non-2xx status. The Code field carries the kebab-case error vocabulary the
// routes emit (e.g. "missing-name", "cap-pool-at-capacity"); callers may
// inspect it to branch on specific conditions when needed.
type APIError struct {
	Status int
	Code   string
	Body   string
}

func (e *APIError) Error() string {
	if e.Code != "" {
		return fmt.Sprintf("caputchin api error: status=%d code=%s", e.Status, e.Code)
	}
	return fmt.Sprintf("caputchin api error: status=%d body=%s", e.Status, e.Body)
}

// IsNotFound reports whether the error represents a 404 response.
func IsNotFound(err error) bool {
	var apiErr *APIError
	if errors.As(err, &apiErr) {
		return apiErr.Status == http.StatusNotFound
	}
	return false
}

// Get issues a GET request and decodes the JSON response into out.
func (c *Client) Get(ctx context.Context, path string, out any) error {
	return c.do(ctx, http.MethodGet, path, nil, out)
}

// Post issues a POST request with a JSON body and decodes the response.
func (c *Client) Post(ctx context.Context, path string, in, out any) error {
	return c.do(ctx, http.MethodPost, path, in, out)
}

// Patch issues a PATCH request with a JSON body and decodes the response.
func (c *Client) Patch(ctx context.Context, path string, in, out any) error {
	return c.do(ctx, http.MethodPatch, path, in, out)
}

// Delete issues a DELETE request. The response body is discarded.
func (c *Client) Delete(ctx context.Context, path string) error {
	return c.do(ctx, http.MethodDelete, path, nil, nil)
}

func (c *Client) do(ctx context.Context, method, path string, in, out any) error {
	var body io.Reader
	if in != nil {
		buf, err := json.Marshal(in)
		if err != nil {
			return fmt.Errorf("marshal request: %w", err)
		}
		body = bytes.NewReader(buf)
	}

	req, err := http.NewRequestWithContext(ctx, method, c.endpoint+path, body)
	if err != nil {
		return fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+c.token)
	req.Header.Set("User-Agent", c.ua)
	if in != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("http call: %w", err)
	}
	defer resp.Body.Close()

	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("read response: %w", err)
	}

	if resp.StatusCode >= 400 {
		return &APIError{Status: resp.StatusCode, Code: extractErrorCode(raw), Body: string(raw)}
	}

	if out == nil || len(raw) == 0 {
		return nil
	}

	if err := json.Unmarshal(raw, out); err != nil {
		return fmt.Errorf("decode response: %w", err)
	}
	return nil
}

// extractErrorCode pulls the kebab-case error vocabulary out of the JSON
// envelopes the management API emits. Routes return one of two shapes:
//
//	{"error": "missing-name"}
//	{"code": "missing-name"}
//
// We accept either. An empty string is returned when neither key is present.
func extractErrorCode(raw []byte) string {
	var env struct {
		Error string `json:"error"`
		Code  string `json:"code"`
	}
	if err := json.Unmarshal(raw, &env); err != nil {
		return ""
	}
	if env.Error != "" {
		return env.Error
	}
	return env.Code
}
