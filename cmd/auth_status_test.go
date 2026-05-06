package cmd

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ollygarden/ollygarden-cli/internal/config"
)

func TestAuthStatus_NoCreds_Exit3(t *testing.T) {
	t.Setenv(config.ConfigFileEnvVar, filepath.Join(t.TempDir(), "config.yaml"))
	t.Setenv("OLLYGARDEN_API_KEY", "")
	t.Setenv("OLLYGARDEN_CONTEXT", "")
	t.Cleanup(func() {
		authStatusNoProbe = false
		contextName = ""
	})

	_, _, err := executeCommand("auth", "status", "--no-probe")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "No credentials")
}

func TestAuthStatus_FromContext_NoProbe_JSON(t *testing.T) {
	t.Setenv(config.ConfigFileEnvVar, filepath.Join(t.TempDir(), "config.yaml"))
	t.Setenv("OLLYGARDEN_API_KEY", "")
	t.Setenv("OLLYGARDEN_CONTEXT", "")
	t.Cleanup(func() {
		authStatusNoProbe = false
		jsonMode = false
		contextName = ""
	})

	cfg := config.New()
	cfg.CurrentContext = "prod"
	cfg.Contexts["prod"] = &config.Context{Name: "prod", APIURL: "https://api.example.com", APIKey: "og_sk_abc123_aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"}
	require.NoError(t, config.Write(cfg))

	out, _, err := executeCommand("auth", "status", "--no-probe", "--json")
	require.NoError(t, err)

	var env struct {
		Data struct {
			Source    string `json:"source"`
			Context   string `json:"context"`
			APIURL    string `json:"api_url"`
			KeyMasked string `json:"key_masked"`
			Probed    bool   `json:"probed"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal([]byte(out), &env))
	assert.Equal(t, "context", env.Data.Source)
	assert.Equal(t, "prod", env.Data.Context)
	assert.Equal(t, "og_sk_abc123_••••", env.Data.KeyMasked)
	assert.False(t, env.Data.Probed)
}

func TestAuthStatus_EnvWinsOverContext_JSON(t *testing.T) {
	t.Setenv(config.ConfigFileEnvVar, filepath.Join(t.TempDir(), "config.yaml"))
	t.Setenv("OLLYGARDEN_API_KEY", "og_sk_envkey_cccccccccccccccccccccccccccccccc")
	t.Setenv("OLLYGARDEN_CONTEXT", "")
	t.Cleanup(func() {
		authStatusNoProbe = false
		jsonMode = false
		contextName = ""
	})

	cfg := config.New()
	cfg.CurrentContext = "prod"
	cfg.Contexts["prod"] = &config.Context{Name: "prod", APIURL: "https://api.example.com", APIKey: "og_sk_abc123_aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"}
	require.NoError(t, config.Write(cfg))

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
	assert.Equal(t, "prod", env.Data.Context, "env source still reports the saved context that would have won")
}

func TestAuthStatus_Probe_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/api/v1/organization", r.URL.Path)
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(map[string]any{"data": map[string]any{"name": "Acme Corp"}, "meta": map[string]any{}})
	}))
	t.Cleanup(srv.Close)

	t.Setenv(config.ConfigFileEnvVar, filepath.Join(t.TempDir(), "config.yaml"))
	t.Setenv("OLLYGARDEN_API_KEY", "")
	t.Setenv("OLLYGARDEN_CONTEXT", "")
	t.Cleanup(func() {
		authStatusNoProbe = false
		jsonMode = false
		contextName = ""
	})
	cfg := config.New()
	cfg.CurrentContext = "prod"
	cfg.Contexts["prod"] = &config.Context{Name: "prod", APIURL: srv.URL, APIKey: "og_sk_abc123_aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"}
	require.NoError(t, config.Write(cfg))

	out, _, err := executeCommand("auth", "status", "--json")
	require.NoError(t, err)

	var env struct {
		Data struct {
			Probed       bool   `json:"probed"`
			Organization string `json:"organization"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal([]byte(out), &env))
	assert.True(t, env.Data.Probed)
	assert.Equal(t, "Acme Corp", env.Data.Organization)
}

func TestAuthStatus_APIURLFlagOverridesContextURL(t *testing.T) {
	t.Setenv(config.ConfigFileEnvVar, filepath.Join(t.TempDir(), "config.yaml"))
	t.Setenv("OLLYGARDEN_API_KEY", "")
	t.Setenv("OLLYGARDEN_CONTEXT", "")
	t.Cleanup(func() {
		authStatusNoProbe = false
		jsonMode = false
		contextName = ""
		apiURL = "https://api.ollygarden.cloud" // restore default
		// Cobra reuses the same rootCmd across executeCommand calls, so the
		// persistent flag's Changed state persists between tests. Reset it to
		// avoid leaking into subsequent tests that don't pass --api-url.
		if f := rootCmd.PersistentFlags().Lookup("api-url"); f != nil {
			f.Changed = false
		}
	})

	cfg := config.New()
	cfg.CurrentContext = "prod"
	cfg.Contexts["prod"] = &config.Context{Name: "prod", APIURL: "https://context-url.example.com", APIKey: "og_sk_abc123_aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"}
	require.NoError(t, config.Write(cfg))

	out, _, err := executeCommand("auth", "status", "--no-probe", "--json", "--api-url", "https://flag-override.example.com")
	require.NoError(t, err)

	var env struct {
		Data struct {
			APIURL string `json:"api_url"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal([]byte(out), &env))
	assert.Equal(t, "https://flag-override.example.com", env.Data.APIURL,
		"--api-url must override the context's api-url for status output")
}
