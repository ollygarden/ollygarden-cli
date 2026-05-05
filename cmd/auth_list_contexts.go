package cmd

import (
	"encoding/json"
	"errors"
	"fmt"
	"sort"

	"github.com/ollygarden/ollygarden-cli/internal/auth"
	"github.com/ollygarden/ollygarden-cli/internal/config"
	"github.com/ollygarden/ollygarden-cli/internal/output"
	"github.com/spf13/cobra"
)

var authListContextsCmd = &cobra.Command{
	Use:   "list-contexts",
	Short: "List saved contexts (no keys are shown)",
	Args:  cobra.NoArgs,
	RunE:  runAuthListContexts,
}

func init() {
	authCmd.AddCommand(authListContextsCmd)
}

func runAuthListContexts(cmd *cobra.Command, _ []string) error {
	cfg, err := config.Load()
	if err != nil {
		var ue *config.UnreadableError
		if errors.As(err, &ue) {
			return auth.ErrConfigUnreadable(ue.Path, ue.Err)
		}
		return auth.ErrConfigUnreadable("", err)
	}

	names := make([]string, 0, len(cfg.Contexts))
	for n := range cfg.Contexts {
		names = append(names, n)
	}
	sort.Strings(names)

	if jsonMode {
		entries := make([]map[string]any, 0, len(names))
		for _, n := range names {
			entries = append(entries, map[string]any{
				"name":    n,
				"api_url": cfg.Contexts[n].APIURL,
				"current": n == cfg.CurrentContext,
			})
		}
		raw, _ := json.Marshal(map[string]any{"data": entries, "meta": map[string]any{}})
		fmt.Fprintln(cmd.OutOrStdout(), string(raw))
		return nil
	}

	if quiet {
		return nil
	}

	f := output.New(cmd.OutOrStdout(), cmd.ErrOrStderr(), false, false)
	rows := make([][]string, 0, len(names))
	for _, n := range names {
		marker := ""
		if n == cfg.CurrentContext {
			marker = "*"
		}
		rows = append(rows, []string{marker, n, cfg.Contexts[n].APIURL})
	}
	f.PrintTable([]string{"CURRENT", "NAME", "API URL"}, rows)
	return nil
}
