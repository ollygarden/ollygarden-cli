package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/ollygarden/ollygarden-cli/internal/client"
	"github.com/ollygarden/ollygarden-cli/internal/exitcode"
	"github.com/spf13/cobra"
)

var (
	apiURL   string
	jsonMode bool
	quiet    bool
	version  = "dev"
	commit   = "none"
	date     = "unknown"
)

var rootCmd = &cobra.Command{
	Use:           "ollygarden",
	Short:         "CLI client for the OllyGarden API",
	Version:       "dev",
	SilenceUsage:  true,
	SilenceErrors: true,
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		// Skip auth check for help, version, and root (no subcommand).
		if cmd.Name() == "help" || cmd.Name() == "version" || cmd.Name() == "ollygarden" {
			return nil
		}

		// Validate URL scheme before any network I/O
		if !strings.HasPrefix(apiURL, "http://") && !strings.HasPrefix(apiURL, "https://") {
			return fmt.Errorf("Error: --api-url must include scheme (e.g., https://api.ollygarden.cloud)")
		}

		apiKey := os.Getenv("OLLYGARDEN_API_KEY")
		if apiKey == "" {
			return &AuthError{}
		}
		return nil
	},
}

// AuthError signals a missing API key.
type AuthError struct{}

func (e *AuthError) Error() string {
	return "Error: OLLYGARDEN_API_KEY not set. Export it: export OLLYGARDEN_API_KEY=og_sk_..."
}

func init() {
	defaultURL := "https://api.ollygarden.cloud"
	if envURL := os.Getenv("OLLYGARDEN_API_URL"); envURL != "" {
		defaultURL = envURL
	}

	rootCmd.PersistentFlags().StringVar(&apiURL, "api-url", defaultURL, "Base URL for the API")
	rootCmd.PersistentFlags().BoolVar(&jsonMode, "json", false, "Output raw JSON")
	rootCmd.PersistentFlags().BoolVarP(&quiet, "quiet", "q", false, "Suppress non-essential output")
}

// SetBuildInfo sets the CLI build metadata. Values come from ldflags at release time.
func SetBuildInfo(v, c, d string) {
	version = v
	commit = c
	date = d
	rootCmd.Version = v
	client.SetVersion(v)
}

// NewClient creates an API client from the current global flags.
func NewClient() *client.Client {
	return client.New(apiURL, os.Getenv("OLLYGARDEN_API_KEY"))
}

// Execute runs the root command and exits with the appropriate code.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		code := exitcode.General
		if _, ok := err.(*AuthError); ok {
			fmt.Fprintln(os.Stderr, err.Error())
			code = exitcode.Auth
		} else if apiErr, ok := err.(*client.APIError); ok {
			code = apiErr.ExitCode()
		} else {
			fmt.Fprintln(os.Stderr, err)
		}
		os.Exit(code)
	}
}
