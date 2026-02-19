package client

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/ollygarden/ollygarden-cli/internal/exitcode"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAuthHeaderSet(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "Bearer test-key", r.Header.Get("Authorization"))
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(APIResponse{Data: json.RawMessage(`{}`)})
	}))
	defer srv.Close()

	c := New(srv.URL, "test-key")
	resp, err := c.Get(context.Background(), "/test", nil)
	require.NoError(t, err)
	resp.Body.Close()
}

func TestBaseURLConstruction(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/api/v1/services", r.URL.Path)
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(APIResponse{Data: json.RawMessage(`[]`)})
	}))
	defer srv.Close()

	c := New(srv.URL, "key")
	resp, err := c.Get(context.Background(), "/services", nil)
	require.NoError(t, err)
	resp.Body.Close()
}

func TestParseResponseSuccess(t *testing.T) {
	body := `{"data":{"id":"123"},"meta":{"timestamp":"2026-01-01T00:00:00Z","trace_id":"abc"}}`
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(body))
	}))
	defer srv.Close()

	c := New(srv.URL, "key")
	resp, err := c.Get(context.Background(), "/test", nil)
	require.NoError(t, err)

	apiResp, err := ParseResponse(resp)
	require.NoError(t, err)
	assert.Equal(t, "abc", apiResp.Meta.TraceID)
	assert.JSONEq(t, `{"id":"123"}`, string(apiResp.Data))
}

func TestParseResponseError(t *testing.T) {
	body := `{"error":{"code":"SERVICE_NOT_FOUND","message":"Service not found"},"meta":{"trace_id":"xyz"}}`
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte(body))
	}))
	defer srv.Close()

	c := New(srv.URL, "key")
	resp, err := c.Get(context.Background(), "/services/bad", nil)
	require.NoError(t, err)

	_, err = ParseResponse(resp)
	require.Error(t, err)

	apiErr, ok := err.(*APIError)
	require.True(t, ok)
	assert.Equal(t, 404, apiErr.StatusCode)
	assert.Equal(t, "SERVICE_NOT_FOUND", apiErr.ErrorResponse.Error.Code)
}

func TestExitCodeMapping(t *testing.T) {
	tests := []struct {
		status   int
		expected int
	}{
		{400, exitcode.Usage},
		{401, exitcode.Auth},
		{404, exitcode.NotFound},
		{429, exitcode.RateLimit},
		{500, exitcode.Server},
		{502, exitcode.Server},
		{503, exitcode.Server},
		{418, exitcode.General}, // unmapped
	}

	for _, tt := range tests {
		apiErr := &APIError{StatusCode: tt.status}
		assert.Equal(t, tt.expected, apiErr.ExitCode(), "HTTP %d", tt.status)
	}
}

func TestExitCodeFromError(t *testing.T) {
	assert.Equal(t, exitcode.Success, ExitCodeFromError(nil))
	assert.Equal(t, exitcode.NotFound, ExitCodeFromError(&APIError{StatusCode: 404}))
	assert.Equal(t, exitcode.General, ExitCodeFromError(assert.AnError))
}

func TestPostSendsJSON(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "POST", r.Method)
		assert.Equal(t, "application/json", r.Header.Get("Content-Type"))
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(APIResponse{Data: json.RawMessage(`{}`)})
	}))
	defer srv.Close()

	c := New(srv.URL, "key")
	resp, err := c.Post(context.Background(), "/webhooks", map[string]string{"name": "test"})
	require.NoError(t, err)
	resp.Body.Close()
}

func TestUserAgentSet(t *testing.T) {
	SetVersion("1.0.0")
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "ollygarden-cli/1.0.0", r.Header.Get("User-Agent"))
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(APIResponse{Data: json.RawMessage(`{}`)})
	}))
	defer srv.Close()

	c := New(srv.URL, "key")
	resp, err := c.Get(context.Background(), "/test", nil)
	require.NoError(t, err)
	resp.Body.Close()
}
