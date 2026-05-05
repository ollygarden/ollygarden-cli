package cmd

import (
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/ollygarden/ollygarden-cli/internal/auth"
	"github.com/ollygarden/ollygarden-cli/internal/client"
	"github.com/ollygarden/ollygarden-cli/internal/config"
	"github.com/ollygarden/ollygarden-cli/internal/exitcode"
	"github.com/spf13/cobra"
)

var (
	apiURL      string
	contextName string // bound to --context persistent flag
	jsonMode    bool
	quiet       bool
	version     = "dev"
	commit      = "none"
	date        = "unknown"

	// resolvedCreds is populated by PersistentPreRunE for non-auth commands
	// so NewClient (and any future helper) can read it.
	resolvedCreds auth.Credentials
)

var rootCmd = &cobra.Command{
	Use:           "ollygarden",
	Short:         "CLI client for the OllyGarden API",
	Version:       "dev",
	SilenceUsage:  true,
	SilenceErrors: true,
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		// URL scheme validation runs for every command, including the auth
		// subtree — it's a pure flag check that doesn't need creds. Without
		// this, `ollygarden auth login --api-url api.ollygarden.cloud` would
		// skip the validation and fail later at the HTTP layer with an
		// unhelpful "unsupported protocol scheme" error.
		if apiURL != "" && !strings.HasPrefix(apiURL, "http://") && !strings.HasPrefix(apiURL, "https://") {
			return fmt.Errorf("--api-url must include scheme (e.g., https://api.ollygarden.cloud)")
		}

		if skipAuthResolution(cmd) {
			return nil
		}

		cfg, err := config.Load()
		if err != nil {
			var ue *config.UnreadableError
			if errors.As(err, &ue) {
				return auth.ErrConfigUnreadable(ue.Path, ue.Err)
			}
			return auth.ErrConfigUnreadable("", err)
		}

		// apiURL has a non-empty default, so we can't pass it unconditionally —
		// that would force the default to win over a context's api-url. Only
		// forward it when the user explicitly set --api-url.
		flagAPIURL := ""
		if cmd.Flags().Changed("api-url") {
			flagAPIURL = apiURL
		}

		creds, err := auth.Resolve(auth.ResolveInputs{
			Config:      cfg,
			EnvAPIKey:   os.Getenv("OLLYGARDEN_API_KEY"),
			EnvAPIURL:   os.Getenv("OLLYGARDEN_API_URL"),
			EnvContext:  os.Getenv(config.ContextEnvVar),
			FlagAPIURL:  flagAPIURL,
			FlagContext: contextName,
		})
		if err != nil {
			return err
		}
		resolvedCreds = creds
		// Make the URL available to NewClient via the existing global.
		apiURL = creds.APIURL
		return nil
	},
}

// skipAuthResolution returns true for commands that should not have
// credentials resolved before running: help, version, the bare root, and
// any command in the `auth` subtree (auth login does the resolution
// itself, the others either don't need creds or compute them on demand).
func skipAuthResolution(cmd *cobra.Command) bool {
	name := cmd.Name()
	if name == "help" || name == "version" || name == "ollygarden" {
		return true
	}
	for c := cmd; c != nil; c = c.Parent() {
		if c.Name() == "auth" {
			return true
		}
	}
	return false
}

func init() {
	defaultURL := "https://api.ollygarden.cloud"
	if envURL := os.Getenv("OLLYGARDEN_API_URL"); envURL != "" {
		defaultURL = envURL
	}

	rootCmd.PersistentFlags().StringVar(&apiURL, "api-url", defaultURL, "Base URL for the API")
	rootCmd.PersistentFlags().StringVar(&contextName, "context", "", "Use a specific saved context for this invocation (overrides current-context, OLLYGARDEN_CONTEXT)")
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

// NewClient creates an API client from the resolved credentials. For
// non-auth commands, PersistentPreRunE will have populated resolvedCreds.
// For auth subcommands that call NewClient (auth status --probe), they
// must populate resolvedCreds themselves before calling.
func NewClient() *client.Client {
	if resolvedCreds.APIKey != "" {
		return client.New(resolvedCreds.APIURL, resolvedCreds.APIKey)
	}
	// Fallback for the rare path where resolution was skipped: use env directly.
	return client.New(apiURL, os.Getenv("OLLYGARDEN_API_KEY"))
}

// Execute runs the root command and exits with the appropriate code.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		code := exitcode.General

		var authErr *auth.Error
		var apiErr *client.APIError
		switch {
		case errors.As(err, &authErr):
			fmt.Fprintln(os.Stderr, "Error: "+authErr.Error())
			code = authErr.ExitCode
		case errors.As(err, &apiErr):
			fmt.Fprintln(os.Stderr, apiErr.Error())
			code = apiErr.ExitCode()
		default:
			fmt.Fprintln(os.Stderr, err)
		}
		os.Exit(code)
	}
}
