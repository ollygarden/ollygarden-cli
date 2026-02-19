package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
)

const basePath = "/api/v1"

// version is set by the cmd package at init time.
var version = "dev"

// SetVersion sets the version used in User-Agent header.
func SetVersion(v string) {
	version = v
}

// Client is the HTTP client for the OllyGarden API.
type Client struct {
	baseURL    string
	apiKey     string
	httpClient *http.Client
}

// New creates a new API client.
func New(baseURL, apiKey string) *Client {
	return &Client{
		baseURL:    strings.TrimRight(baseURL, "/") + basePath,
		apiKey:     apiKey,
		httpClient: &http.Client{},
	}
}

// Get performs a GET request.
func (c *Client) Get(ctx context.Context, path string, query url.Values) (*http.Response, error) {
	u := c.baseURL + path
	if len(query) > 0 {
		u += "?" + query.Encode()
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return nil, err
	}
	return c.do(req)
}

// Post performs a POST request with a JSON body.
func (c *Client) Post(ctx context.Context, path string, body any) (*http.Response, error) {
	return c.doWithBody(ctx, http.MethodPost, path, body)
}

// Put performs a PUT request with a JSON body.
func (c *Client) Put(ctx context.Context, path string, body any) (*http.Response, error) {
	return c.doWithBody(ctx, http.MethodPut, path, body)
}

// Delete performs a DELETE request.
func (c *Client) Delete(ctx context.Context, path string) (*http.Response, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodDelete, c.baseURL+path, nil)
	if err != nil {
		return nil, err
	}
	return c.do(req)
}

func (c *Client) doWithBody(ctx context.Context, method, path string, body any) (*http.Response, error) {
	var buf bytes.Buffer
	if body != nil {
		if err := json.NewEncoder(&buf).Encode(body); err != nil {
			return nil, fmt.Errorf("encoding request body: %w", err)
		}
	}
	req, err := http.NewRequestWithContext(ctx, method, c.baseURL+path, &buf)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	return c.do(req)
}

func (c *Client) do(req *http.Request) (*http.Response, error) {
	req.Header.Set("Authorization", "Bearer "+c.apiKey)
	req.Header.Set("User-Agent", "ollygarden-cli/"+version)
	return c.httpClient.Do(req)
}

// Response types matching the API envelope.

type APIResponse struct {
	Data  json.RawMessage `json:"data"`
	Meta  ResponseMeta    `json:"meta"`
	Links json.RawMessage `json:"links,omitempty"`
}

type ResponseMeta struct {
	Timestamp string `json:"timestamp,omitempty"`
	Total     int    `json:"total,omitempty"`
	HasMore   bool   `json:"has_more,omitempty"`
	TraceID   string `json:"trace_id,omitempty"`
}

type ErrorResponse struct {
	Error ErrorDetail  `json:"error"`
	Meta  ResponseMeta `json:"meta"`
}

type ErrorDetail struct {
	Code    string            `json:"code"`
	Message string            `json:"message"`
	Details map[string]string `json:"details,omitempty"`
}

// ParseResponse reads the response body and returns a parsed APIResponse or an *APIError.
func ParseResponse(resp *http.Response) (*APIResponse, error) {
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("reading response body: %w", err)
	}

	if resp.StatusCode >= 400 {
		var errResp ErrorResponse
		if jsonErr := json.Unmarshal(body, &errResp); jsonErr != nil {
			// Can't parse error body — return raw status
			return nil, &APIError{StatusCode: resp.StatusCode}
		}
		return nil, &APIError{StatusCode: resp.StatusCode, ErrorResponse: &errResp}
	}

	var apiResp APIResponse
	if err := json.Unmarshal(body, &apiResp); err != nil {
		return nil, fmt.Errorf("parsing response: %w", err)
	}
	return &apiResp, nil
}
