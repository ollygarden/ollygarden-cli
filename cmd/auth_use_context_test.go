package cmd

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ollygarden/ollygarden-cli/internal/config"
)

func setupTwoContextsForUse(t *testing.T) {
	t.Helper()
	t.Setenv(config.ConfigFileEnvVar, filepath.Join(t.TempDir(), "config.yaml"))
	t.Setenv("OLLYGARDEN_API_KEY", "")
	t.Setenv("OLLYGARDEN_CONTEXT", "")
	cfg := config.New()
	cfg.CurrentContext = "prod"
	cfg.Contexts["prod"] = &config.Context{Name: "prod", APIURL: "https://prod", APIKey: "og_sk_prod00_aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"}
	cfg.Contexts["dev"] = &config.Context{Name: "dev", APIURL: "https://dev", APIKey: "og_sk_dev000_bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"}
	require.NoError(t, config.Write(cfg))
	t.Cleanup(func() {
		contextName = ""
		jsonMode = false
		quiet = false
	})
}

func TestAuthUseContext_Switches(t *testing.T) {
	setupTwoContextsForUse(t)
	_, _, err := executeCommand("auth", "use-context", "dev")
	require.NoError(t, err)
	cfg, _ := config.Load()
	assert.Equal(t, "dev", cfg.CurrentContext)
}

func TestAuthUseContext_NotFound(t *testing.T) {
	setupTwoContextsForUse(t)
	_, _, err := executeCommand("auth", "use-context", "ghost")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}
