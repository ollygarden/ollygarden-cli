package cmd

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ollygarden/ollygarden-cli/internal/config"
)

func setupAuthLoginEnv(t *testing.T) *httptest.Server {
	t.Helper()
	dir := t.TempDir()
	t.Setenv(config.ConfigFileEnvVar, filepath.Join(dir, "config.yaml"))
	t.Setenv("OLLYGARDEN_API_KEY", "")
	t.Setenv("OLLYGARDEN_CONTEXT", "")

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/api/v1/organization", r.URL.Path)
		assert.True(t, strings.HasPrefix(r.Header.Get("Authorization"), "Bearer og_sk_"))
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"data": map[string]any{"name": "Acme Corp"},
			"meta": map[string]any{},
		})
	}))
	t.Cleanup(srv.Close)

	// Reset auth_login flags between tests since they're package globals.
	t.Cleanup(func() {
		authLoginTokenFile = ""
		authLoginNoActivate = false
		jsonMode = false
		quiet = false
		contextName = "" // prevents --context value leaking into other tests
		apiURL = "https://api.ollygarden.cloud"
		// Cobra reuses the same rootCmd across executeCommand calls, so the
		// persistent flag's Changed state persists between tests. Reset it so
		// subsequent tests that don't pass --api-url aren't affected.
		if f := rootCmd.PersistentFlags().Lookup("api-url"); f != nil {
			f.Changed = false
		}
	})
	return srv
}

func TestAuthLogin_TokenFile_HappyPath(t *testing.T) {
	srv := setupAuthLoginEnv(t)
	tokenPath := filepath.Join(t.TempDir(), "token")
	require.NoError(t, os.WriteFile(tokenPath, []byte("og_sk_abc123_aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa\n"), 0o600))

	out, _, err := executeCommand("auth", "login",
		"--api-url", srv.URL,
		"--context", "prod",
		"--token-file", tokenPath,
	)
	require.NoError(t, err)
	assert.Contains(t, out+"", "") // touch out to silence linter
	cfg, err := config.Load()
	require.NoError(t, err)
	require.NotNil(t, cfg.Contexts["prod"])
	assert.Equal(t, "og_sk_abc123_aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", cfg.Contexts["prod"].APIKey)
	assert.Equal(t, "prod", cfg.CurrentContext)
}

func TestAuthLogin_TokenFile_JSONOutput(t *testing.T) {
	srv := setupAuthLoginEnv(t)
	tokenPath := filepath.Join(t.TempDir(), "token")
	require.NoError(t, os.WriteFile(tokenPath, []byte("og_sk_abc123_aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"), 0o600))

	out, _, err := executeCommand("auth", "login",
		"--api-url", srv.URL,
		"--context", "prod",
		"--token-file", tokenPath,
		"--json",
	)
	require.NoError(t, err)

	var env struct {
		Data struct {
			Context      string `json:"context"`
			Organization string `json:"organization"`
			KeyMasked    string `json:"key_masked"`
			Activated    bool   `json:"activated"`
			APIURL       string `json:"api_url"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal([]byte(out), &env))
	assert.Equal(t, "prod", env.Data.Context)
	assert.Equal(t, "Acme Corp", env.Data.Organization)
	assert.Equal(t, "og_sk_abc123_••••", env.Data.KeyMasked)
	assert.True(t, env.Data.Activated)
}

func TestAuthLogin_TokenFile_Missing(t *testing.T) {
	srv := setupAuthLoginEnv(t)
	_, _, err := executeCommand("auth", "login",
		"--api-url", srv.URL,
		"--context", "prod",
		"--token-file", "/nonexistent/path/token",
	)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "Cannot read token file")
}

func TestAuthLogin_TokenRejected(t *testing.T) {
	dir := t.TempDir()
	t.Setenv(config.ConfigFileEnvVar, filepath.Join(dir, "config.yaml"))
	t.Setenv("OLLYGARDEN_API_KEY", "")
	t.Setenv("OLLYGARDEN_CONTEXT", "")

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	}))
	t.Cleanup(srv.Close)
	t.Cleanup(func() {
		authLoginTokenFile = ""
		authLoginNoActivate = false
		jsonMode = false
		quiet = false
		contextName = ""                        // prevents --context value leaking into other tests
		apiURL = "https://api.ollygarden.cloud" // restore default
		// Cobra reuses the same rootCmd across executeCommand calls, so the
		// persistent flag's Changed state persists between tests. Reset it so
		// subsequent tests that don't pass --api-url aren't affected.
		if f := rootCmd.PersistentFlags().Lookup("api-url"); f != nil {
			f.Changed = false
		}
	})

	tokenPath := filepath.Join(t.TempDir(), "token")
	require.NoError(t, os.WriteFile(tokenPath, []byte("og_sk_abc123_aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"), 0o600))

	_, _, err := executeCommand("auth", "login",
		"--api-url", srv.URL,
		"--context", "prod",
		"--token-file", tokenPath,
	)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "Token rejected")
}

func TestAuthLogin_InvalidTokenFormat(t *testing.T) {
	srv := setupAuthLoginEnv(t)
	tokenPath := filepath.Join(t.TempDir(), "token")
	require.NoError(t, os.WriteFile(tokenPath, []byte("not-a-real-token"), 0o600))

	_, _, err := executeCommand("auth", "login",
		"--api-url", srv.URL,
		"--context", "prod",
		"--token-file", tokenPath,
	)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "Invalid token format")
}

func TestAuthLogin_NoActivate(t *testing.T) {
	srv := setupAuthLoginEnv(t)
	// Pre-seed
	pre := config.New()
	pre.CurrentContext = "existing"
	pre.Contexts["existing"] = &config.Context{Name: "existing", APIURL: "https://x", APIKey: "og_sk_pre000_eeeeeeeeeeeeeeeeeeeeeeeeeeeeeeee"}
	require.NoError(t, config.Write(pre))

	tokenPath := filepath.Join(t.TempDir(), "token")
	require.NoError(t, os.WriteFile(tokenPath, []byte("og_sk_new000_aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"), 0o600))

	_, _, err := executeCommand("auth", "login",
		"--api-url", srv.URL,
		"--context", "new",
		"--token-file", tokenPath,
		"--no-activate",
	)
	require.NoError(t, err)

	cfg, _ := config.Load()
	assert.Equal(t, "existing", cfg.CurrentContext, "current-context must not change with --no-activate")
	assert.NotNil(t, cfg.Contexts["new"], "new context must still be added")
}
