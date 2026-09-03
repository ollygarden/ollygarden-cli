package cmd

import (
	"encoding/json"
	"fmt"

	"github.com/ollygarden/ollygarden-cli/internal/client"
	"github.com/ollygarden/ollygarden-cli/internal/output"
	"github.com/spf13/cobra"
)

type roseRepositoryActivationRequest struct {
	IsActive bool `json:"is_active"`
}

type roseRepositoryActivationResult struct {
	ID              string `json:"id"`
	IsActive        bool   `json:"is_active"`
	ActiveRepoCount int    `json:"active_repo_count"`
	RepoLimit       *int   `json:"repo_limit"`
}

var roseRepositoriesActivateCmd = &cobra.Command{
	Use:   "activate <repository-id>",
	Short: "Activate Rose analysis for a repository",
	Args:  cobra.ExactArgs(1),
	RunE:  runRoseRepositoriesActivate,
}

func init() {
	roseRepositoriesCmd.AddCommand(roseRepositoriesActivateCmd)
}

func runRoseRepositoriesActivate(cmd *cobra.Command, args []string) error {
	f := newFormatter(cmd)
	if err := validateRoseRepositoryID(f, args[0]); err != nil {
		return err
	}
	apiResp, err := rosePatch(cmd.Context(), f, "/rose/repositories/"+args[0], roseRepositoryActivationRequest{IsActive: true})
	if err != nil {
		return err
	}
	return printRoseRepositoryActivation(cmd, f, apiResp)
}

func printRoseRepositoryActivation(cmd *cobra.Command, f *output.Formatter, response *client.APIResponse) error {
	if f.IsJSON() {
		raw, _ := json.Marshal(response)
		f.PrintJSON(raw)
		return nil
	}
	if f.IsQuiet() {
		return nil
	}
	var result roseRepositoryActivationResult
	if err := json.Unmarshal(response.Data, &result); err != nil {
		return fmt.Errorf("parsing repository activation data: %w", err)
	}
	state := "inactive"
	if result.IsActive {
		state = "active"
	}
	limit := "unlimited"
	if result.RepoLimit != nil {
		limit = fmt.Sprint(*result.RepoLimit)
	}
	fmt.Fprintf(cmd.OutOrStdout(), "Repository %s is %s (%d/%s active).\n", result.ID, state, result.ActiveRepoCount, limit)
	return nil
}
