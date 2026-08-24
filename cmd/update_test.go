package cmd

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/ollygarden/ollygarden-cli/internal/selfupdate"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupUpdateCommand(t *testing.T, result selfupdate.Result, updateErr error) *selfupdate.Options {
	t.Helper()
	previousUpdater := runSelfUpdater
	var received selfupdate.Options
	runSelfUpdater = func(_ context.Context, options selfupdate.Options) (selfupdate.Result, error) {
		received = options
		if options.Progress != nil {
			options.Progress("Checking for updates...")
		}
		return result, updateErr
	}
	SetBuildInfo("1.0.0", "test", "2026-08-24T00:00:00Z")
	jsonMode = false
	quiet = false
	updateForce = false
	t.Setenv("OLLYGARDEN_API_KEY", "")
	t.Setenv("OLLYGARDEN_CONFIG", t.TempDir()+"/config.yaml")
	t.Cleanup(func() {
		runSelfUpdater = previousUpdater
		SetBuildInfo("dev", "none", "unknown")
		jsonMode = false
		quiet = false
		updateForce = false
		for _, flag := range []string{"json", "quiet"} {
			if f := rootCmd.PersistentFlags().Lookup(flag); f != nil {
				f.Changed = false
				_ = f.Value.Set("false")
			}
		}
		if f := updateCmd.Flags().Lookup("force"); f != nil {
			f.Changed = false
			_ = f.Value.Set("false")
		}
	})
	return &received
}

func TestUpdateInstallsNewerRelease(t *testing.T) {
	received := setupUpdateCommand(t, selfupdate.Result{
		CurrentVersion: "1.0.0",
		LatestVersion:  "1.1.0",
		Executable:     "/tmp/ollygarden",
		Updated:        true,
	}, nil)
	out, stderr, err := executeCommand("update")
	require.NoError(t, err)
	assert.Equal(t, "Updated ollygarden from v1.0.0 to v1.1.0.\n", out)
	assert.Contains(t, stderr, "Checking for updates")
	assert.Equal(t, "1.0.0", received.CurrentVersion)
	assert.False(t, received.Force)
}

func TestUpdateAlreadyCurrent(t *testing.T) {
	setupUpdateCommand(t, selfupdate.Result{CurrentVersion: "1.0.0", LatestVersion: "1.0.0"}, nil)
	out, _, err := executeCommand("update")
	require.NoError(t, err)
	assert.Equal(t, "ollygarden is already up to date (v1.0.0).\n", out)
}

func TestUpdateForceReinstallsCurrentRelease(t *testing.T) {
	received := setupUpdateCommand(t, selfupdate.Result{
		CurrentVersion: "1.0.0",
		LatestVersion:  "1.0.0",
		Updated:        true,
		Forced:         true,
	}, nil)
	out, _, err := executeCommand("update", "--force")
	require.NoError(t, err)
	assert.Equal(t, "Reinstalled ollygarden v1.0.0.\n", out)
	assert.True(t, received.Force)
}

func TestUpdateJSONOutput(t *testing.T) {
	setupUpdateCommand(t, selfupdate.Result{
		CurrentVersion: "1.0.0",
		LatestVersion:  "1.1.0",
		Executable:     "/tmp/ollygarden",
		Updated:        true,
	}, nil)
	out, stderr, err := executeCommand("update", "--json")
	require.NoError(t, err)
	assert.Empty(t, stderr)
	var envelope struct {
		Data selfupdate.Result `json:"data"`
		Meta map[string]any    `json:"meta"`
	}
	require.NoError(t, json.Unmarshal([]byte(out), &envelope))
	assert.True(t, envelope.Data.Updated)
	assert.Equal(t, "1.1.0", envelope.Data.LatestVersion)
	assert.NotNil(t, envelope.Meta)
}

func TestUpdateQuietSuppressesOutput(t *testing.T) {
	setupUpdateCommand(t, selfupdate.Result{
		CurrentVersion: "1.0.0",
		LatestVersion:  "1.1.0",
		Updated:        true,
	}, nil)
	out, stderr, err := executeCommand("update", "--quiet")
	require.NoError(t, err)
	assert.Empty(t, out)
	assert.Empty(t, stderr)
}

func TestUpdateJSONErrorIsReportedOnce(t *testing.T) {
	setupUpdateCommand(t, selfupdate.Result{}, errors.New("network unavailable"))
	_, stderr, err := executeCommand("update", "--json")
	require.Error(t, err)
	var envelope struct {
		Error struct {
			Code    string `json:"code"`
			Message string `json:"message"`
		} `json:"error"`
	}
	require.NoError(t, json.Unmarshal([]byte(stderr), &envelope))
	assert.Equal(t, "UPDATE_FAILED", envelope.Error.Code)
	assert.Contains(t, envelope.Error.Message, "network unavailable")
	assert.Equal(t, 1, handleRootErr(err, rootCmd.ErrOrStderr()))
}

func TestUpdateHumanError(t *testing.T) {
	setupUpdateCommand(t, selfupdate.Result{}, errors.New("permission denied"))
	_, _, err := executeCommand("update")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "Error: updating ollygarden")
	assert.Contains(t, err.Error(), "permission denied")
}

func TestUpdateDoesNotRequireAPIAuth(t *testing.T) {
	setupUpdateCommand(t, selfupdate.Result{CurrentVersion: "1.0.0", LatestVersion: "1.0.0"}, nil)
	_, _, err := executeCommand("update")
	require.NoError(t, err)
}

func TestUpdateRejectsArguments(t *testing.T) {
	setupUpdateCommand(t, selfupdate.Result{}, nil)
	_, _, err := executeCommand("update", "unexpected")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unknown command")
}
