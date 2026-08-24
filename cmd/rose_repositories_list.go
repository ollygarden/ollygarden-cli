package cmd

import (
	"encoding/json"
	"fmt"
	"strconv"

	"github.com/spf13/cobra"
)

type roseFindingCounts struct {
	Critical   int `json:"critical"`
	High       int `json:"high"`
	Medium     int `json:"medium"`
	Low        int `json:"low"`
	Suggestion int `json:"suggestion"`
}

type roseInventoryRepo struct {
	ID                  string            `json:"id"`
	RepoFullName        string            `json:"repo_full_name"`
	IsActive            bool              `json:"is_active"`
	LastScannedAt       *string           `json:"last_scanned_at"`
	ActiveFindingsCount int               `json:"active_findings_count"`
	FindingCounts       roseFindingCounts `json:"finding_counts"`
}

type roseInventoryInstallation struct {
	Repos []roseInventoryRepo `json:"repos"`
}

var roseRepositoriesListCmd = &cobra.Command{
	Use:   "list",
	Short: "List repositories connected to Rose",
	Args:  cobra.NoArgs,
	RunE:  runRoseRepositoriesList,
}

func init() {
	roseRepositoriesCmd.AddCommand(roseRepositoriesListCmd)
}

func runRoseRepositoriesList(cmd *cobra.Command, args []string) error {
	f := newFormatter(cmd)

	apiResp, err := roseGet(cmd.Context(), f, "/rose/repositories", nil)
	if err != nil {
		return err
	}

	if f.IsJSON() {
		raw, _ := json.Marshal(apiResp)
		f.PrintJSON(raw)
		return nil
	}
	if f.IsQuiet() {
		return nil
	}

	var list roseListEnvelope
	if err := json.Unmarshal(apiResp.Data, &list); err != nil {
		return fmt.Errorf("parsing repositories data: %w", err)
	}
	var installations []roseInventoryInstallation
	if err := json.Unmarshal(list.Data, &installations); err != nil {
		return fmt.Errorf("parsing repositories data: %w", err)
	}

	// Flatten the installation nesting: the repo rows are what matter in a
	// terminal. Installation metadata stays available via --json.
	headers := []string{"ID", "REPOSITORY", "ACTIVE", "LAST SCANNED", "FINDINGS", "CRIT/HIGH/MED/LOW"}
	var rows [][]string
	for _, inst := range installations {
		for _, repo := range inst.Repos {
			counts := repo.FindingCounts
			rows = append(rows, []string{
				repo.ID,
				repo.RepoFullName,
				boolYesNo(repo.IsActive),
				roseTime(repo.LastScannedAt),
				strconv.Itoa(repo.ActiveFindingsCount),
				fmt.Sprintf("%d/%d/%d/%d", counts.Critical, counts.High, counts.Medium, counts.Low),
			})
		}
	}
	f.PrintTable(headers, rows)
	return nil
}
