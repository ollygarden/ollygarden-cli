package cmd

import (
	"encoding/json"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ollygarden/ollygarden-cli/internal/config"
)

func setupThreeContextsForList(t *testing.T) {
	t.Helper()
	t.Setenv(config.ConfigFileEnvVar, filepath.Join(t.TempDir(), "config.yaml"))
	t.Setenv("OLLYGARDEN_API_KEY", "")
	t.Setenv("OLLYGARDEN_CONTEXT", "")
	cfg := config.New()
	cfg.CurrentContext = "prod"
	cfg.Contexts["prod"] = &config.Context{Name: "prod", APIURL: "https://prod", APIKey: "og_sk_prod00_aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"}
	cfg.Contexts["dev"] = &config.Context{Name: "dev", APIURL: "https://dev", APIKey: "og_sk_dev000_bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"}
	cfg.Contexts["staging"] = &config.Context{Name: "staging", APIURL: "https://staging", APIKey: "og_sk_stg000_cccccccccccccccccccccccccccccccc"}
	require.NoError(t, config.Write(cfg))
	t.Cleanup(func() {
		contextName = ""
		jsonMode = false
		quiet = false
	})
}

func TestAuthListContexts_Human(t *testing.T) {
	setupThreeContextsForList(t)
	out, _, err := executeCommand("auth", "list-contexts")
	require.NoError(t, err)

	// Sorted alphabetically; current marker on prod.
	assert.Contains(t, out, "CURRENT")
	assert.Contains(t, out, "NAME")
	assert.Contains(t, out, "API URL")
	assert.Contains(t, out, "*")
	assert.Contains(t, out, "prod")
	assert.Contains(t, out, "dev")
	assert.Contains(t, out, "staging")
	// No keys printed.
	assert.False(t, strings.Contains(out, "og_sk_prod00"), "keys must never appear in list-contexts")
}

func TestAuthListContexts_JSON(t *testing.T) {
	setupThreeContextsForList(t)
	out, _, err := executeCommand("auth", "list-contexts", "--json")
	require.NoError(t, err)

	var env struct {
		Data []struct {
			Name    string `json:"name"`
			APIURL  string `json:"api_url"`
			Current bool   `json:"current"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal([]byte(out), &env))
	require.Len(t, env.Data, 3)
	for _, e := range env.Data {
		if e.Name == "prod" {
			assert.True(t, e.Current)
		} else {
			assert.False(t, e.Current)
		}
	}
}
