package cmd

import (
	"encoding/json"
	"fmt"
	"strconv"

	"github.com/spf13/cobra"
)

var magnoliaFindingsOrgID string

type magnoliaFindingsArtifact struct {
	Run struct {
		OrganizationID string `json:"organization_id"`
		RunID          string `json:"run_id"`
	} `json:"run"`
	Findings []json.RawMessage `json:"findings"`
	Groups   []struct {
		Title      string   `json:"title"`
		Severity   string   `json:"severity"`
		Pillar     string   `json:"pillar"`
		FindingIDs []string `json:"finding_ids"`
	} `json:"groups"`
}

var magnoliaFindingsCmd = &cobra.Command{
	Use:   "findings",
	Short: "Get the latest Magnolia findings artifact",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		f := newFormatter(cmd)
		body, err := magnoliaGet(cmd, f, "/magnolia/findings", magnoliaFindingsOrgID)
		if err != nil {
			return err
		}
		if f.IsJSON() {
			f.PrintJSON(body)
			return nil
		}
		if f.IsQuiet() {
			return nil
		}
		var artifact magnoliaFindingsArtifact
		if err := json.Unmarshal(body, &artifact); err != nil {
			return fmt.Errorf("parsing Magnolia findings: %w", err)
		}
		rows := make([][]string, len(artifact.Groups))
		for i, group := range artifact.Groups {
			rows[i] = []string{group.Severity, group.Pillar, group.Title, strconv.Itoa(len(group.FindingIDs))}
		}
		f.PrintTable([]string{"SEVERITY", "PILLAR", "GROUP", "FINDINGS"}, rows)
		return nil
	},
}

func init() {
	magnoliaCmd.AddCommand(magnoliaFindingsCmd)
	magnoliaFindingsCmd.Flags().StringVar(&magnoliaFindingsOrgID, "org-id", "", "Organization identifier")
	_ = magnoliaFindingsCmd.MarkFlagRequired("org-id")
}
