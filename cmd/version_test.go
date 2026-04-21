package cmd

import (
	"encoding/json"
	"runtime"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestVersionHumanOutput(t *testing.T) {
	SetBuildInfo("1.2.3", "abc1234", "2026-04-21T10:00:00Z")
	defer SetBuildInfo("dev", "none", "unknown")

	out, _, err := executeCommand("version")
	require.NoError(t, err)

	assert.Contains(t, out, "Version:")
	assert.Contains(t, out, "1.2.3")
	assert.Contains(t, out, "Commit:")
	assert.Contains(t, out, "abc1234")
	assert.Contains(t, out, "Built:")
	assert.Contains(t, out, "2026-04-21T10:00:00Z")
	assert.Contains(t, out, runtime.Version())
	assert.Contains(t, out, runtime.GOOS)
	assert.Contains(t, out, runtime.GOARCH)
}

func TestVersionJSONOutput(t *testing.T) {
	SetBuildInfo("1.2.3", "abc1234", "2026-04-21T10:00:00Z")
	defer SetBuildInfo("dev", "none", "unknown")
	jsonMode = true
	defer func() { jsonMode = false }()

	out, _, err := executeCommand("version")
	require.NoError(t, err)

	var info versionInfo
	require.NoError(t, json.Unmarshal([]byte(out), &info))
	assert.Equal(t, "1.2.3", info.Version)
	assert.Equal(t, "abc1234", info.Commit)
	assert.Equal(t, "2026-04-21T10:00:00Z", info.Date)
	assert.Equal(t, runtime.Version(), info.Go)
	assert.Equal(t, runtime.GOOS, info.OS)
	assert.Equal(t, runtime.GOARCH, info.Arch)
}

func TestVersionNoAuthRequired(t *testing.T) {
	t.Setenv("OLLYGARDEN_API_KEY", "")
	_, _, err := executeCommand("version")
	assert.NoError(t, err)
}

func TestVersionDefaults(t *testing.T) {
	assert.Equal(t, "dev", version)
	assert.Equal(t, "none", commit)
	assert.Equal(t, "unknown", date)
}
