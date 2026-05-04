package cmd

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ollygarden/ollygarden-cli/internal/config"
)

func setupTwoContexts(t *testing.T) {
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
		authLogoutContext = ""
		authLogoutAll = false
		authLogoutConfirm = false
		contextName = ""
	})
}

func TestAuthLogout_DefaultRemovesCurrent(t *testing.T) {
	setupTwoContexts(t)
	_, _, err := executeCommand("auth", "logout")
	require.NoError(t, err)

	cfg, _ := config.Load()
	assert.Empty(t, cfg.CurrentContext, "current-context must be unset")
	_, gone := cfg.Contexts["prod"]
	assert.False(t, gone, "prod context must be removed")
	_, kept := cfg.Contexts["dev"]
	assert.True(t, kept, "dev context must remain")
}

func TestAuthLogout_NamedContext(t *testing.T) {
	setupTwoContexts(t)
	_, _, err := executeCommand("auth", "logout", "--context", "dev")
	require.NoError(t, err)

	cfg, _ := config.Load()
	_, gone := cfg.Contexts["dev"]
	assert.False(t, gone)
	assert.Equal(t, "prod", cfg.CurrentContext, "current-context unchanged when removing non-current")
}

func TestAuthLogout_ContextNotFound(t *testing.T) {
	setupTwoContexts(t)
	_, _, err := executeCommand("auth", "logout", "--context", "ghost")
	require.Error(t, err)
	assert.Contains(t, err.Error(), `not found`)
}

func TestAuthLogout_AllRequiresConfirm(t *testing.T) {
	setupTwoContexts(t)
	// Non-TTY, no --confirm: must refuse.
	_, _, err := executeCommand("auth", "logout", "--all")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "--confirm")
}

func TestAuthLogout_AllWithConfirm(t *testing.T) {
	setupTwoContexts(t)
	_, _, err := executeCommand("auth", "logout", "--all", "--confirm")
	require.NoError(t, err)

	cfg, _ := config.Load()
	assert.Empty(t, cfg.Contexts, "all contexts removed")
	assert.Empty(t, cfg.CurrentContext)
}
