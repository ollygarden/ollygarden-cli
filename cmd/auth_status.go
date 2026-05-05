package cmd

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/ollygarden/ollygarden-cli/internal/auth"
	"github.com/ollygarden/ollygarden-cli/internal/config"
	"github.com/spf13/cobra"
)

var authStatusNoProbe bool

var authStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show the active credential and verify it works",
	Long: `Print the active credential's source, URL, and a masked key.

By default, makes a single GET /api/v1/organization request to confirm
the token is still accepted (matches ` + "`gh auth status`" + ` precedent). Pass
--no-probe to skip the network call.

Exit codes:
  0  Logged in (and probe succeeded if probing).
  3  No credential is configured, or the probe got 401.`,
	Args: cobra.NoArgs,
	RunE: runAuthStatus,
}

func init() {
	authStatusCmd.Flags().BoolVar(&authStatusNoProbe, "no-probe", false, "Skip the /organization probe")
	authCmd.AddCommand(authStatusCmd)
}

func runAuthStatus(cmd *cobra.Command, _ []string) error {
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

	probed := false
	orgName := ""
	if !authStatusNoProbe {
		probed = true
		ctx, cancel := context.WithTimeout(cmd.Context(), 30*time.Second)
		defer cancel()
		orgName, err = probeOrgFromCmd(ctx, creds.APIURL, creds.APIKey)
		if err != nil {
			return err
		}
	}

	source := "context"
	if creds.Source == auth.SourceEnv {
		source = "env"
	}

	if jsonMode {
		out := map[string]any{
			"data": map[string]any{
				"source":     source,
				"context":    creds.ContextName,
				"api_url":    creds.APIURL,
				"key_masked": auth.MaskKey(creds.APIKey),
				"probed":     probed,
			},
			"meta": map[string]any{},
		}
		if probed && orgName != "" {
			out["data"].(map[string]any)["organization"] = orgName
		}
		raw, _ := json.Marshal(out)
		fmt.Fprintln(cmd.OutOrStdout(), string(raw))
		return nil
	}

	if quiet {
		return nil
	}

	srcLine := source
	switch source {
	case "env":
		srcLine = "env (OLLYGARDEN_API_KEY)"
		if creds.ContextName != "" {
			srcLine += fmt.Sprintf(" — overrides saved context %q", creds.ContextName)
		}
	case "context":
		srcLine = fmt.Sprintf("context: %s", creds.ContextName)
	}

	fmt.Fprintf(cmd.OutOrStdout(), "Source:        %s\n", srcLine)
	fmt.Fprintf(cmd.OutOrStdout(), "API URL:       %s\n", creds.APIURL)
	fmt.Fprintf(cmd.OutOrStdout(), "Key:           %s\n", auth.MaskKey(creds.APIKey))
	if probed && orgName != "" {
		fmt.Fprintf(cmd.OutOrStdout(), "Organization:  %s\n", orgName)
	}
	return nil
}

// probeOrgFromCmd is a small wrapper that mirrors auth.probeOrganization
// behavior — separated here so cmd doesn't have to import the unexported
// version. Returns auth.ErrTokenRejected on 401.
func probeOrgFromCmd(ctx context.Context, baseURL, token string) (string, error) {
	url := strings.TrimRight(baseURL, "/") + "/api/v1/organization"
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return "", fmt.Errorf("building request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Accept", "application/json")

	resp, err := (&http.Client{Timeout: 30 * time.Second}).Do(req)
	if err != nil {
		return "", fmt.Errorf("calling %s: %w", url, err)
	}
	defer resp.Body.Close()

	switch resp.StatusCode {
	case http.StatusOK:
		var envelope struct {
			Data struct {
				Name string `json:"name"`
			} `json:"data"`
		}
		_ = json.NewDecoder(resp.Body).Decode(&envelope)
		return envelope.Data.Name, nil
	case http.StatusUnauthorized:
		return "", auth.ErrTokenRejected()
	default:
		return "", fmt.Errorf("unexpected status %d from %s", resp.StatusCode, url)
	}
}
