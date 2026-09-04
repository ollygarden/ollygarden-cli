package cmd

import (
	"encoding/json"
	"fmt"
	"strconv"

	"github.com/spf13/cobra"
)

var legacyMagnoliaFindingsOrgID string

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

var analyticsFindingsCmd = &cobra.Command{
	Use:   "findings",
	Short: "Get the latest Magnolia findings artifact",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runAnalyticsFindings(cmd, "")
	},
}

var legacyMagnoliaFindingsCmd = &cobra.Command{
	Use:    "findings",
	Short:  "Get the latest Magnolia findings artifact",
	Args:   cobra.NoArgs,
	Hidden: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		if legacyMagnoliaFindingsOrgID == "" {
			return fmt.Errorf("--org-id is required")
		}
		return runAnalyticsFindings(cmd, legacyMagnoliaFindingsOrgID)
	},
}

func runAnalyticsFindings(cmd *cobra.Command, orgID string) error {
	f := newFormatter(cmd)
	body, err := magnoliaGet(cmd, f, "/magnolia/findings", orgID)
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
}

func init() {
	analyticsCmd.AddCommand(analyticsFindingsCmd)
	magnoliaCmd.AddCommand(legacyMagnoliaFindingsCmd)
	legacyMagnoliaFindingsCmd.Flags().StringVar(&legacyMagnoliaFindingsOrgID, "org-id", "", "Organization identifier")
	_ = legacyMagnoliaFindingsCmd.MarkFlagRequired("org-id")
}
