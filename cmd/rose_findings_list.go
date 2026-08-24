package cmd

import (
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"

	"github.com/spf13/cobra"
)

var (
	roseFindingsListSeverity    string
	roseFindingsListCategory    string
	roseFindingsListStatus      string
	roseFindingsListExecutionID string
	roseFindingsListPage        int
	roseFindingsListLimit       int
)

type roseFindingListItem struct {
	FindingID    string  `json:"finding_id"`
	RepositoryID string  `json:"repository_id"`
	RepoFullName string  `json:"repo_full_name"`
	Severity     string  `json:"severity"`
	Category     string  `json:"category"`
	Title        string  `json:"title"`
	DisplayTitle *string `json:"display_title"`
	PRNumber     *int    `json:"pr_number"`
	PRStatus     *string `json:"pr_status"`
}

var roseFindingsListCmd = &cobra.Command{
	Use:   "list",
	Short: "List codebase findings across the organization",
	Args:  cobra.NoArgs,
	RunE:  runRoseFindingsList,
}

func init() {
	roseFindingsCmd.AddCommand(roseFindingsListCmd)
	roseFindingsListCmd.Flags().StringVar(&roseFindingsListSeverity, "severity", "", "Filter by severity (comma-separated: critical, high, medium, low, suggestion)")
	roseFindingsListCmd.Flags().StringVar(&roseFindingsListCategory, "category", "", "Filter by category (comma-separated, e.g. \"Sensitive Data,Volume\")")
	roseFindingsListCmd.Flags().StringVar(&roseFindingsListStatus, "status", "active", "Finding status (active, resolved, all)")
	roseFindingsListCmd.Flags().StringVar(&roseFindingsListExecutionID, "execution-id", "", "Only findings produced by this execution")
	roseFindingsListCmd.Flags().IntVar(&roseFindingsListPage, "page", 1, "Page number (≥1)")
	roseFindingsListCmd.Flags().IntVar(&roseFindingsListLimit, "limit", 50, "Results per page (1-100)")
}

func runRoseFindingsList(cmd *cobra.Command, args []string) error {
	f := newFormatter(cmd)

	if roseFindingsListLimit < 1 || roseFindingsListLimit > 100 {
		return roseInvalidParameters(f, "--limit must be between 1 and 100")
	}
	if roseFindingsListPage < 1 {
		return roseInvalidParameters(f, "--page must be >= 1")
	}
	switch roseFindingsListStatus {
	case "active", "resolved", "all":
	default:
		return roseInvalidParameters(f, "--status must be one of: active, resolved, all")
	}
	if !roseCSVValuesAllowed(roseFindingsListSeverity, "critical", "high", "medium", "low", "suggestion") {
		return roseInvalidParameters(f, "--severity must contain only: critical, high, medium, low, suggestion")
	}
	if roseFindingsListExecutionID != "" && !roseUUIDPattern.MatchString(roseFindingsListExecutionID) {
		return roseInvalidParameters(f, "--execution-id must be a UUID")
	}

	query := url.Values{}
	query.Set("page", strconv.Itoa(roseFindingsListPage))
	query.Set("page_size", strconv.Itoa(roseFindingsListLimit))
	query.Set("status", roseFindingsListStatus)
	if roseFindingsListSeverity != "" {
		query.Set("severity", roseFindingsListSeverity)
	}
	if roseFindingsListCategory != "" {
		query.Set("category", roseFindingsListCategory)
	}
	if roseFindingsListExecutionID != "" {
		query.Set("executionId", roseFindingsListExecutionID)
	}

	apiResp, err := roseGet(cmd.Context(), f, "/rose/findings", query)
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
		return fmt.Errorf("parsing findings data: %w", err)
	}
	var findings []roseFindingListItem
	if err := json.Unmarshal(list.Data, &findings); err != nil {
		return fmt.Errorf("parsing findings data: %w", err)
	}

	headers := []string{"FINDING ID", "SEVERITY", "CATEGORY", "REPOSITORY", "PR", "TITLE"}
	rows := make([][]string, len(findings))
	for i, fd := range findings {
		title := fd.Title
		if fd.DisplayTitle != nil && *fd.DisplayTitle != "" {
			title = *fd.DisplayTitle
		}
		rows[i] = []string{fd.FindingID, fd.Severity, orDash(fd.Category), fd.RepoFullName, prRef(fd.PRNumber, fd.PRStatus), title}
	}
	f.PrintTable(headers, rows)

	if list.Pagination.HasMore {
		f.PrintPageHint(list.Pagination.Total, roseFindingsListPage, roseFindingsListLimit)
	}
	return nil
}
