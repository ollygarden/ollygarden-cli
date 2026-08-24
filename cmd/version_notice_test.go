package cmd

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/ollygarden/ollygarden-cli/internal/selfupdate"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupVersionNoticeTest(t *testing.T) *cobra.Command {
	t.Helper()
	previousCheck := checkLatestVersion
	previousTerminal := stdoutIsTerminal
	previousWait := versionNoticeWait
	command := &cobra.Command{
		Use:  "notice-test",
		Args: cobra.NoArgs,
		RunE: func(*cobra.Command, []string) error { return nil },
	}
	rootCmd.AddCommand(command)
	SetBuildInfo("1.0.0", "test", "2026-08-24T00:00:00Z")
	stdoutIsTerminal = func() bool { return true }
	versionNoticeWait = 50 * time.Millisecond
	jsonMode = false
	quiet = false
	t.Setenv("OLLYGARDEN_API_KEY", "og_sk_notice_aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa")
	t.Setenv("OLLYGARDEN_CONFIG", t.TempDir()+"/config.yaml")
	t.Cleanup(func() {
		rootCmd.RemoveCommand(command)
		checkLatestVersion = previousCheck
		stdoutIsTerminal = previousTerminal
		versionNoticeWait = previousWait
		versionCheck = nil
		SetBuildInfo("dev", "none", "unknown")
		jsonMode = false
		quiet = false
	})
	return command
}

func TestVersionNoticeShownAfterSuccessfulInteractiveCommand(t *testing.T) {
	setupVersionNoticeTest(t)
	var checkedVersion string
	checkLatestVersion = func(_ context.Context, current string) (*selfupdate.Notice, error) {
		checkedVersion = current
		return &selfupdate.Notice{
			Version: "1.1.0",
			URL:     "https://github.com/ollygarden/ollygarden-cli/releases/tag/v1.1.0",
		}, nil
	}

	out, stderr, err := executeCommand("notice-test")
	require.NoError(t, err)
	assert.Empty(t, out)
	assert.Equal(t, "1.0.0", checkedVersion)
	assert.Equal(t, "\nUpdate Available\nNew version v1.1.0 is available. Run `ollygarden update`.\n\nChangelog: https://github.com/ollygarden/ollygarden-cli/releases/tag/v1.1.0\n", stderr)
}

func TestVersionNoticeChecksEveryEligibleRun(t *testing.T) {
	setupVersionNoticeTest(t)
	checks := 0
	checkLatestVersion = func(context.Context, string) (*selfupdate.Notice, error) {
		checks++
		return nil, nil
	}

	_, _, err := executeCommand("notice-test")
	require.NoError(t, err)
	_, _, err = executeCommand("notice-test")
	require.NoError(t, err)
	assert.Equal(t, 2, checks)
}

func TestVersionNoticeSilentlyIgnoresCheckFailure(t *testing.T) {
	setupVersionNoticeTest(t)
	checkLatestVersion = func(context.Context, string) (*selfupdate.Notice, error) {
		return nil, errors.New("network unavailable")
	}

	_, stderr, err := executeCommand("notice-test")
	require.NoError(t, err)
	assert.Empty(t, stderr)
}

func TestVersionNoticeDoesNotDelayPastGracePeriod(t *testing.T) {
	setupVersionNoticeTest(t)
	versionNoticeWait = time.Millisecond
	release := make(chan struct{})
	checkLatestVersion = func(context.Context, string) (*selfupdate.Notice, error) {
		<-release
		return &selfupdate.Notice{Version: "1.1.0", URL: "https://example.com"}, nil
	}

	_, stderr, err := executeCommand("notice-test")
	close(release)
	require.NoError(t, err)
	assert.Empty(t, stderr)
}

func TestVersionNoticeEligibility(t *testing.T) {
	command := setupVersionNoticeTest(t)
	assert.True(t, shouldCheckVersion(command))

	SetBuildInfo("dev", "none", "unknown")
	assert.False(t, shouldCheckVersion(command))
	SetBuildInfo("1.0.0", "test", "2026-08-24T00:00:00Z")

	jsonMode = true
	assert.False(t, shouldCheckVersion(command))
	jsonMode = false

	quiet = true
	assert.False(t, shouldCheckVersion(command))
	quiet = false

	stdoutIsTerminal = func() bool { return false }
	assert.False(t, shouldCheckVersion(command))
	stdoutIsTerminal = func() bool { return true }

	assert.False(t, shouldCheckVersion(rootCmd))
	assert.False(t, shouldCheckVersion(updateCmd))
	assert.False(t, shouldCheckVersion(versionCmd))

	completion := &cobra.Command{Use: "completion"}
	completionChild := &cobra.Command{Use: "bash"}
	completion.AddCommand(completionChild)
	rootCmd.AddCommand(completion)
	t.Cleanup(func() { rootCmd.RemoveCommand(completion) })
	assert.False(t, shouldCheckVersion(completionChild))
}
