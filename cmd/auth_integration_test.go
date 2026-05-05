//go:build integration

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

func newOrgServerInt(t *testing.T) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"data": map[string]any{"name": "Acme Corp"},
			"meta": map[string]any{},
		})
	}))
	t.Cleanup(srv.Close)
	return srv
}

// Scenario 1: login → status reports the context → roundtrip persists token.
func TestIntegration_LoginThenStatus(t *testing.T) {
	srv := newOrgServerInt(t)
	t.Setenv(config.ConfigFileEnvVar, filepath.Join(t.TempDir(), "config.yaml"))
	t.Setenv("OLLYGARDEN_API_KEY", "")
	t.Setenv("OLLYGARDEN_CONTEXT", "")
	t.Cleanup(func() {
		authLoginTokenFile = ""
		authLoginNoActivate = false
		authStatusNoProbe = false
		jsonMode = false
		contextName = ""
	})

	tokenPath := filepath.Join(t.TempDir(), "token")
	require.NoError(t, writeFile(tokenPath, "og_sk_int000_aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"))

	_, _, err := executeCommand("auth", "login",
		"--api-url", srv.URL,
		"--context", "prod",
		"--token-file", tokenPath,
	)
	require.NoError(t, err)

	out, _, err := executeCommand("auth", "status", "--no-probe", "--json")
	require.NoError(t, err)

	var env struct {
		Data struct {
			Source  string `json:"source"`
			Context string `json:"context"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal([]byte(out), &env))
	assert.Equal(t, "context", env.Data.Source)
	assert.Equal(t, "prod", env.Data.Context)
}

// Scenario 2: env var wins; status reports both env source and the would-have-won context.
func TestIntegration_EnvWinsButContextNoted(t *testing.T) {
	srv := newOrgServerInt(t)
	t.Setenv(config.ConfigFileEnvVar, filepath.Join(t.TempDir(), "config.yaml"))
	t.Setenv("OLLYGARDEN_API_KEY", "")
	t.Setenv("OLLYGARDEN_CONTEXT", "")
	t.Cleanup(func() {
		authLoginTokenFile = ""
		authLoginNoActivate = false
		authStatusNoProbe = false
		jsonMode = false
		contextName = ""
	})

	tokenPath := filepath.Join(t.TempDir(), "token")
	require.NoError(t, writeFile(tokenPath, "og_sk_int000_aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"))
	_, _, err := executeCommand("auth", "login",
		"--api-url", srv.URL, "--context", "prod", "--token-file", tokenPath,
	)
	require.NoError(t, err)

	t.Setenv("OLLYGARDEN_API_KEY", "og_sk_envvar_dddddddddddddddddddddddddddddddd")
	out, _, err := executeCommand("auth", "status", "--no-probe", "--json")
	require.NoError(t, err)

	var env struct {
		Data struct {
			Source  string `json:"source"`
			Context string `json:"context"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal([]byte(out), &env))
	assert.Equal(t, "env", env.Data.Source)
	assert.Equal(t, "prod", env.Data.Context)
}

// Scenario 3: two logins + use-context produce a stable file with both contexts.
func TestIntegration_TwoLoginsThenUseContext(t *testing.T) {
	srv := newOrgServerInt(t)
	t.Setenv(config.ConfigFileEnvVar, filepath.Join(t.TempDir(), "config.yaml"))
	t.Setenv("OLLYGARDEN_API_KEY", "")
	t.Setenv("OLLYGARDEN_CONTEXT", "")
	t.Cleanup(func() {
		authLoginTokenFile = ""
		authLoginNoActivate = false
		jsonMode = false
		contextName = ""
	})

	tokenPathProd := filepath.Join(t.TempDir(), "tprod")
	tokenPathDev := filepath.Join(t.TempDir(), "tdev")
	require.NoError(t, writeFile(tokenPathProd, "og_sk_int001_aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"))
	require.NoError(t, writeFile(tokenPathDev, "og_sk_int002_bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"))

	_, _, err := executeCommand("auth", "login",
		"--api-url", srv.URL, "--context", "prod", "--token-file", tokenPathProd,
	)
	require.NoError(t, err)

	_, _, err = executeCommand("auth", "login",
		"--api-url", srv.URL, "--context", "dev", "--token-file", tokenPathDev,
	)
	require.NoError(t, err)

	_, _, err = executeCommand("auth", "use-context", "prod")
	require.NoError(t, err)

	out, _, err := executeCommand("auth", "list-contexts")
	require.NoError(t, err)
	assert.True(t, strings.Contains(out, "prod"))
	assert.True(t, strings.Contains(out, "dev"))
}

func writeFile(path, contents string) error {
	return os.WriteFile(path, []byte(contents), 0o600)
}
