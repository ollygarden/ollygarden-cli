package cmd

import (
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/ollygarden/ollygarden-cli/internal/client"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const createdAPIKeySecret = "og_sk_Ab3xYz_0123456789abcdef0123456789abcdef"

func setupAPIKeysCreateServer(t *testing.T, handler http.HandlerFunc) {
	t.Helper()
	resetHelpFlag(t, apiKeysCreateCmd)
	oldDescription := apiKeysCreateDescription
	setupAPIServer(t, handler)
	t.Cleanup(func() {
		apiKeysCreateDescription = oldDescription
	})
}

func apiKeysCreateResponse() string {
	return `{"data":{"id":"key-new","key":"` + createdAPIKeySecret + `","key_preview":"og_sk_***def",` +
		`"description":"CI telemetry","created_at":"2026-09-06T12:00:00Z","last_seen_at":null,` +
		`"expires_at":null,"is_active":true,"created_by":"api-key:parent"},` +
		`"meta":{"timestamp":"2026-09-06T12:00:00Z","trace_id":"trace-create"}}`
}

func TestAPIKeysCreateHuman(t *testing.T) {
	setupAPIKeysCreateServer(t, func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPost, r.Method)
		assert.Equal(t, "/api/v1/api-keys", r.URL.Path)
		var body apiKeyCreateRequest
		require.NoError(t, json.NewDecoder(r.Body).Decode(&body))
		assert.Equal(t, "CI telemetry", body.Description)
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte(apiKeysCreateResponse()))
	})

	out, stderr, err := executeCommand("api-keys", "create", "--description", "  CI telemetry  ")
	require.NoError(t, err)
	assert.Contains(t, out, "key-new")
	assert.Contains(t, out, "CI telemetry")
	assert.Contains(t, out, createdAPIKeySecret)
	assert.Contains(t, stderr, "Save this API key now")
	assert.Contains(t, stderr, "cannot be retrieved again")
	assert.NotContains(t, stderr, createdAPIKeySecret)
}

func TestAPIKeysCreateJSON(t *testing.T) {
	setupAPIKeysCreateServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte(apiKeysCreateResponse()))
	})

	out, stderr, err := executeCommand("api-keys", "create", "--description", "CI telemetry", "--json")
	require.NoError(t, err)
	assert.Empty(t, stderr)

	var envelope client.APIResponse
	require.NoError(t, json.Unmarshal([]byte(out), &envelope))
	assert.Equal(t, "trace-create", envelope.Meta.TraceID)
	assert.Contains(t, string(envelope.Data), createdAPIKeySecret)
}

func TestAPIKeysCreateQuiet(t *testing.T) {
	setupAPIKeysCreateServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte(apiKeysCreateResponse()))
	})

	out, stderr, err := executeCommand("api-keys", "create", "--description", "CI telemetry", "--quiet")
	require.NoError(t, err)
	assert.Empty(t, out)
	assert.Empty(t, stderr)
}

func TestAPIKeysCreateRejectsInvalidDescriptionBeforeRequest(t *testing.T) {
	for _, test := range []struct {
		name        string
		description string
	}{
		{name: "missing"},
		{name: "blank", description: "   "},
		{name: "too long", description: strings.Repeat("é", 101)},
		{name: "invalid UTF-8", description: string([]byte{0xff})},
	} {
		t.Run(test.name, func(t *testing.T) {
			setupAPIKeysCreateServer(t, func(w http.ResponseWriter, r *http.Request) {
				t.Fatal("request must not be sent for an invalid description")
			})

			args := []string{"api-keys", "create"}
			if test.description != "" {
				args = append(args, "--description", test.description)
			}
			_, stderr, err := executeCommand(args...)
			require.Error(t, err)
			apiErr, ok := err.(*client.APIError)
			require.True(t, ok)
			assert.Equal(t, 2, apiErr.ExitCode())
			assert.Contains(t, stderr, "--description must be")
		})
	}
}

func TestAPIKeysCreateInvalidDescriptionJSON(t *testing.T) {
	setupAPIKeysCreateServer(t, func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("request must not be sent for an invalid description")
	})

	_, stderr, err := executeCommand("api-keys", "create", "--description", " ", "--json")
	require.Error(t, err)
	var envelope client.ErrorResponse
	require.NoError(t, json.Unmarshal([]byte(stderr), &envelope))
	assert.Equal(t, "INVALID_PARAMETERS", envelope.Error.Code)
	assert.Equal(t, "--description must be between 1 and 100 characters", envelope.Error.Message)
}

func TestAPIKeysCreateSendsOnlyDescription(t *testing.T) {
	setupAPIKeysCreateServer(t, func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		require.NoError(t, err)
		assert.JSONEq(t, `{"description":"CI telemetry"}`, string(body))
		assert.NotContains(t, string(body), "organization")
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte(apiKeysCreateResponse()))
	})

	_, _, err := executeCommand("api-keys", "create", "--description", "CI telemetry")
	require.NoError(t, err)
}

func TestAPIKeysCreateRejectsSuccessWithoutOneTimeKey(t *testing.T) {
	setupAPIKeysCreateServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte(`{"data":{"id":"key-new","description":"CI telemetry"},"meta":{}}`))
	})

	out, stderr, err := executeCommand("api-keys", "create", "--description", "CI telemetry", "--json")
	require.EqualError(t, err, "created API key response did not include the one-time key")
	assert.Empty(t, out)
	assert.Empty(t, stderr)
}

func TestAPIKeysCreateAPIErrors(t *testing.T) {
	for _, test := range []struct {
		status int
		code   string
		exit   int
	}{
		{status: http.StatusBadRequest, code: "INVALID_PARAMETERS", exit: 2},
		{status: http.StatusUnauthorized, code: "INVALID_API_KEY", exit: 3},
		{status: http.StatusNotFound, code: "NOT_FOUND", exit: 4},
		{status: http.StatusTooManyRequests, code: "RATE_LIMIT_EXCEEDED", exit: 5},
		{status: http.StatusInternalServerError, code: "INTERNAL_ERROR", exit: 6},
	} {
		t.Run(test.code, func(t *testing.T) {
			setupAPIKeysCreateServer(t, func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(test.status)
				_, _ = w.Write([]byte(`{"error":{"code":"` + test.code + `","message":"request failed"},"meta":{"trace_id":"trace-error"}}`))
			})

			_, stderr, err := executeCommand("api-keys", "create", "--description", "CI telemetry")
			require.Error(t, err)
			apiErr, ok := err.(*client.APIError)
			require.True(t, ok)
			assert.Equal(t, test.exit, apiErr.ExitCode())
			assert.Contains(t, stderr, "request failed")
			assert.NotContains(t, stderr, "og_sk_")
		})
	}
}

func TestAPIKeysCreateHelp(t *testing.T) {
	out, _, err := executeCommand("api-keys", "create", "--help")
	require.NoError(t, err)
	assert.Contains(t, out, "Create an API key")
	assert.Contains(t, out, "--description")
}
