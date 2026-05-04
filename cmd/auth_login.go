package cmd

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"net/url"
	"os"
	"strings"

	"github.com/ollygarden/ollygarden-cli/internal/auth"
	"github.com/spf13/cobra"
	"golang.org/x/term"
)

var (
	authLoginTokenFile  string
	authLoginNoActivate bool
)

const tokenURLHint = "Get a token at https://ollygarden.app/settings"

var authLoginCmd = &cobra.Command{
	Use:   "login",
	Short: "Save credentials to disk for a context",
	Long: `Save an OllyGarden API key to a named context on disk.

Three ways to provide the token, picked in this order:
  1. --token-file PATH        read from a file
  2. stdin (when piped)       read one line
  3. interactive TTY          prompt with hidden input

The token is validated against the API before any data is written. On
success, the context becomes the current-context unless --no-activate
is passed.`,
	Args: cobra.NoArgs,
	RunE: runAuthLogin,
}

func init() {
	authLoginCmd.Flags().StringVar(&authLoginTokenFile, "token-file", "", "Read the API token from this file path")
	authLoginCmd.Flags().BoolVar(&authLoginNoActivate, "no-activate", false, "Save the context without setting it as current-context")
	authCmd.AddCommand(authLoginCmd)
}

func runAuthLogin(cmd *cobra.Command, _ []string) error {
	token, err := readTokenInput(cmd)
	if err != nil {
		return err
	}

	resolvedURL := apiURL // default, env, or --api-url all flow through here
	ctxName := contextName
	if ctxName == "" {
		derived, derr := deriveContextName(resolvedURL)
		if derr != nil {
			return fmt.Errorf("deriving context name: %w", derr)
		}
		ctxName = derived
	}

	result, err := auth.Login(cmd.Context(), auth.LoginInputs{
		Token:       token,
		APIURL:      resolvedURL,
		ContextName: ctxName,
		Activate:    !authLoginNoActivate,
	})
	if err != nil {
		return err
	}

	if jsonMode {
		out := map[string]any{
			"data": map[string]any{
				"context":      result.ContextName,
				"api_url":      result.APIURL,
				"organization": result.OrganizationName,
				"key_masked":   result.KeyMasked,
				"activated":    result.Activated,
			},
			"meta": map[string]any{},
		}
		raw, _ := json.Marshal(out)
		fmt.Fprintln(cmd.OutOrStdout(), string(raw))
		return nil
	}

	if quiet {
		return nil
	}

	orgPart := ""
	if result.OrganizationName != "" {
		orgPart = fmt.Sprintf(" to %q", result.OrganizationName)
	}
	fmt.Fprintf(cmd.OutOrStdout(),
		"✓ Logged in%s as context %q (%s).\n",
		orgPart, result.ContextName, result.APIURL,
	)
	return nil
}

// readTokenInput selects the token source per the spec's precedence:
// --token-file > non-TTY stdin > TTY prompt.
func readTokenInput(cmd *cobra.Command) (string, error) {
	if authLoginTokenFile != "" {
		data, err := os.ReadFile(authLoginTokenFile)
		if err != nil {
			return "", auth.ErrTokenFileNotFound(authLoginTokenFile)
		}
		return strings.TrimSpace(string(data)), nil
	}

	stdin := cmd.InOrStdin()
	if !isTerminal(stdin) {
		raw, err := io.ReadAll(stdin)
		if err != nil {
			return "", fmt.Errorf("reading token from stdin: %w", err)
		}
		// Take only the first line, trimmed.
		s := bufio.NewScanner(strings.NewReader(string(raw)))
		if s.Scan() {
			return strings.TrimSpace(s.Text()), nil
		}
		return "", fmt.Errorf("no token on stdin")
	}

	// Interactive TTY: print hint to stderr, then ReadPassword.
	fmt.Fprintln(cmd.ErrOrStderr(), tokenURLHint)
	fmt.Fprint(cmd.ErrOrStderr(), "Paste your OllyGarden API key: ")
	tokBytes, err := term.ReadPassword(int(os.Stdin.Fd()))
	fmt.Fprintln(cmd.ErrOrStderr())
	if err != nil {
		return "", fmt.Errorf("reading token: %w", err)
	}
	return strings.TrimSpace(string(tokBytes)), nil
}

// isTerminal reports whether r is connected to a terminal. Anything that
// isn't *os.File (e.g. bytes.Buffer in tests) counts as non-TTY.
func isTerminal(r io.Reader) bool {
	f, ok := r.(*os.File)
	if !ok {
		return false
	}
	return term.IsTerminal(int(f.Fd()))
}

// deriveContextName implements the spec rule: strip leading "api." from the
// hostname, replace remaining "." with "-".
func deriveContextName(rawURL string) (string, error) {
	u, err := url.Parse(rawURL)
	if err != nil {
		return "", err
	}
	host := u.Hostname()
	host = strings.TrimPrefix(host, "api.")
	host = strings.ReplaceAll(host, ".", "-")
	if host == "" {
		return "default", nil
	}
	return host, nil
}
