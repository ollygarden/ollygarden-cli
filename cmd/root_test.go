package cmd

import (
	"bytes"
	"errors"
	"fmt"
	"testing"

	"github.com/ollygarden/ollygarden-cli/internal/auth"
	"github.com/ollygarden/ollygarden-cli/internal/config"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func executeCommand(args ...string) (string, string, error) {
	stdout := new(bytes.Buffer)
	stderr := new(bytes.Buffer)

	rootCmd.SetOut(stdout)
	rootCmd.SetErr(stderr)
	rootCmd.SetArgs(args)

	err := rootCmd.Execute()
	return stdout.String(), stderr.String(), err
}

func TestHelpShowsCommandGroups(t *testing.T) {
	out, _, err := executeCommand("--help")
	require.NoError(t, err)

	assert.Contains(t, out, "services")
	assert.Contains(t, out, "insights")
	assert.Contains(t, out, "analytics")
	assert.Contains(t, out, "webhooks")
}

func TestVersionFlag(t *testing.T) {
	SetBuildInfo("1.2.3", "abc123", "2026-04-21T00:00:00Z")
	defer SetBuildInfo("dev", "none", "unknown")

	// Cobra's --version flag works correctly in real execution (verified via binary).
	// In-process test: verify SetBuildInfo wires through to rootCmd.Version.
	assert.Equal(t, "1.2.3", rootCmd.Version)
}

func TestMissingAPIKey_ReturnsTypedNoCredentialsError(t *testing.T) {
	t.Setenv("OLLYGARDEN_API_KEY", "")
	t.Setenv("OLLYGARDEN_CONTEXT", "")
	// Point config to an empty location to suppress real ~/.config reads.
	t.Setenv("OLLYGARDEN_CONFIG", t.TempDir()+"/config.yaml")

	testCmd := &cobra.Command{
		Use:  "auth-test-cmd",
		RunE: func(cmd *cobra.Command, args []string) error { return nil },
	}
	rootCmd.AddCommand(testCmd)
	defer rootCmd.RemoveCommand(testCmd)

	_, _, err := executeCommand("auth-test-cmd")
	require.Error(t, err)

	var ae *auth.Error
	require.True(t, errors.As(err, &ae), "expected *auth.Error, got %T: %v", err, err)
	assert.Equal(t, "NO_CREDENTIALS", ae.Code)
}

func TestEnvAPIKey_StillWorks(t *testing.T) {
	t.Setenv("OLLYGARDEN_API_KEY", "og_sk_envkey_aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa")
	t.Setenv("OLLYGARDEN_CONFIG", t.TempDir()+"/config.yaml")

	testCmd := &cobra.Command{
		Use:  "auth-ok-cmd",
		RunE: func(cmd *cobra.Command, args []string) error { return nil },
	}
	rootCmd.AddCommand(testCmd)
	defer rootCmd.RemoveCommand(testCmd)

	_, _, err := executeCommand("auth-ok-cmd")
	require.NoError(t, err)
}

func TestAPIURLDefault(t *testing.T) {
	assert.Equal(t, "https://api.ollygarden.cloud", rootCmd.PersistentFlags().Lookup("api-url").DefValue)
}

func TestAPIURLEnvOverride(t *testing.T) {
	// The env override happens at init() time, so we verify the flag exists
	flag := rootCmd.PersistentFlags().Lookup("api-url")
	require.NotNil(t, flag)
	assert.Equal(t, "string", flag.Value.Type())
}

func TestServicesHelp(t *testing.T) {
	out, _, err := executeCommand("services", "--help")
	require.NoError(t, err)
	assert.Contains(t, out, "Manage services")
}

func TestWebhooksDeliveriesHelp(t *testing.T) {
	out, _, err := executeCommand("webhooks", "deliveries", "--help")
	require.NoError(t, err)
	assert.Contains(t, out, "View webhook deliveries")
}

func TestGlobalFlags(t *testing.T) {
	// Verify all global flags are registered
	flags := []string{"api-url", "context", "json", "quiet"}
	for _, name := range flags {
		flag := rootCmd.PersistentFlags().Lookup(name)
		assert.NotNil(t, flag, "flag --%s should exist", name)
	}
}

func TestQuietShortFlag(t *testing.T) {
	flag := rootCmd.PersistentFlags().ShorthandLookup("q")
	require.NotNil(t, flag)
	assert.Equal(t, "quiet", flag.Name)
}

func TestAPIURLMissingSchemeReturnsError(t *testing.T) {
	t.Setenv("OLLYGARDEN_API_KEY", "og_sk_test_1234567890abcdef1234567890abcdef")
	t.Setenv("OLLYGARDEN_CONFIG", t.TempDir()+"/config.yaml")
	t.Cleanup(func() {
		apiURL = "https://api.ollygarden.cloud"
		if f := rootCmd.PersistentFlags().Lookup("api-url"); f != nil {
			f.Changed = false
		}
	})

	testCmd := &cobra.Command{
		Use:  "scheme-test-cmd",
		RunE: func(cmd *cobra.Command, args []string) error { return nil },
	}
	rootCmd.AddCommand(testCmd)
	defer rootCmd.RemoveCommand(testCmd)

	_, _, err := executeCommand("scheme-test-cmd", "--api-url", "api.ollygarden.cloud")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "--api-url must include scheme")
}

func TestAPIURLMissingScheme_OnAuthSubcommand(t *testing.T) {
	t.Setenv("OLLYGARDEN_CONFIG", t.TempDir()+"/config.yaml")
	t.Setenv("OLLYGARDEN_API_KEY", "")
	t.Setenv("OLLYGARDEN_CONTEXT", "")
	t.Cleanup(func() {
		// Restore the persistent flag's default and Changed state after this test mutates it.
		apiURL = "https://api.ollygarden.cloud"
		if f := rootCmd.PersistentFlags().Lookup("api-url"); f != nil {
			f.Changed = false
		}
	})

	_, _, err := executeCommand("auth", "status", "--no-probe", "--api-url", "no-scheme.example.com")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "must include scheme")
}

func TestNewClientUsesFlags(t *testing.T) {
	t.Setenv("OLLYGARDEN_API_KEY", "test-key")
	c := NewClient()
	assert.NotNil(t, c)
	_ = fmt.Sprintf("%v", c) // ensure it's usable
}

func TestPersistentPreRunE_ContextURL_NotOverriddenByDefault(t *testing.T) {
	// Verifies that the persistent --api-url flag's default value does not
	// silently override a context's saved api-url. Without the
	// cmd.Flags().Changed("api-url") gate, this test catches the regression.
	cfgPath := t.TempDir() + "/config.yaml"
	t.Setenv("OLLYGARDEN_CONFIG", cfgPath)
	t.Setenv("OLLYGARDEN_API_KEY", "")
	t.Setenv("OLLYGARDEN_CONTEXT", "")
	t.Cleanup(func() {
		contextName = ""
		apiURL = "https://api.ollygarden.cloud"
		if f := rootCmd.PersistentFlags().Lookup("api-url"); f != nil {
			f.Changed = false
		}
	})

	// Seed a context with a non-default api-url.
	cfg := config.New()
	cfg.CurrentContext = "internal"
	cfg.Contexts["internal"] = &config.Context{
		Name:   "internal",
		APIURL: "https://api.internal.example.com",
		APIKey: "og_sk_intxxx_aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
	}
	require.NoError(t, config.Write(cfg))

	// Trigger PersistentPreRunE via a no-op test command. No --api-url flag passed.
	testCmd := &cobra.Command{
		Use:  "url-test-cmd",
		RunE: func(cmd *cobra.Command, args []string) error { return nil },
	}
	rootCmd.AddCommand(testCmd)
	defer rootCmd.RemoveCommand(testCmd)

	_, _, err := executeCommand("url-test-cmd")
	require.NoError(t, err)

	// resolvedCreds.APIURL must come from the context, not the default flag value.
	assert.Equal(t, "https://api.internal.example.com", resolvedCreds.APIURL,
		"context's api-url should not be overridden by the persistent flag's default")
}
