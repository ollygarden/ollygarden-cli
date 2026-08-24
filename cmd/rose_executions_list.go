package cmd

import (
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"

	"github.com/spf13/cobra"
)

var (
	roseExecutionsListLimit        int
	roseExecutionsListOffset       int
	roseExecutionsListStatus       string
	roseExecutionsListRepositoryID string
	roseExecutionsListType         string
)

type roseExecutionPhase struct {
	Key         string  `json:"key"`
	Label       string  `json:"label"`
	State       string  `json:"state"`
	StartedAt   *string `json:"startedAt"`
	CompletedAt *string `json:"completedAt"`
}

type roseExecutionSummary struct {
	ID            string  `json:"id"`
	ExecutionType string  `json:"executionType"`
	Status        *string `json:"status"`
	TriggerSource *string `json:"triggerSource"`
	Ref           string  `json:"ref"`
	CommitSHA     string  `json:"commitSha"`
	StartedAt     string  `json:"startedAt"`
	CompletedAt   *string `json:"completedAt"`
	RepositoryID  string  `json:"repositoryId"`
	RepoOwner     string  `json:"repoOwner"`
	RepoName      string  `json:"repoName"`
}

var roseExecutionsListCmd = &cobra.Command{
	Use:   "list",
	Short: "List Rose executions",
	Args:  cobra.NoArgs,
	RunE:  runRoseExecutionsList,
}

func init() {
	roseExecutionsCmd.AddCommand(roseExecutionsListCmd)
	roseExecutionsListCmd.Flags().IntVar(&roseExecutionsListLimit, "limit", 50, "Maximum number of results (1-100)")
	roseExecutionsListCmd.Flags().IntVar(&roseExecutionsListOffset, "offset", 0, "Number of results to skip (≥0)")
	roseExecutionsListCmd.Flags().StringVar(&roseExecutionsListStatus, "status", "", "Filter by status (pending, running, completed, failed)")
	roseExecutionsListCmd.Flags().StringVar(&roseExecutionsListRepositoryID, "repository-id", "", "Filter by repository ID")
	roseExecutionsListCmd.Flags().StringVar(&roseExecutionsListType, "type", "", "Filter by execution type (comma-separated: review, fix, instrumentation, deliveryhero-migrate-execute)")
}

func runRoseExecutionsList(cmd *cobra.Command, args []string) error {
	if roseExecutionsListLimit < 1 || roseExecutionsListLimit > 100 {
		return fmt.Errorf("--limit must be between 1 and 100")
	}
	if roseExecutionsListOffset < 0 {
		return fmt.Errorf("--offset must be >= 0")
	}

	f := newFormatter(cmd)

	query := url.Values{}
	query.Set("limit", strconv.Itoa(roseExecutionsListLimit))
	query.Set("offset", strconv.Itoa(roseExecutionsListOffset))
	if roseExecutionsListStatus != "" {
		query.Set("status", roseExecutionsListStatus)
	}
	if roseExecutionsListRepositoryID != "" {
		query.Set("repositoryId", roseExecutionsListRepositoryID)
	}
	if roseExecutionsListType != "" {
		query.Set("executionType", roseExecutionsListType)
	}

	apiResp, err := roseGet(cmd.Context(), f, "/rose/executions", query)
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
		return fmt.Errorf("parsing executions data: %w", err)
	}
	var executions []roseExecutionSummary
	if err := json.Unmarshal(list.Data, &executions); err != nil {
		return fmt.Errorf("parsing executions data: %w", err)
	}

	headers := []string{"ID", "TYPE", "STATUS", "REPOSITORY", "STARTED", "COMPLETED"}
	rows := make([][]string, len(executions))
	for i, ex := range executions {
		started := ex.StartedAt
		rows[i] = []string{
			ex.ID,
			ex.ExecutionType,
			strOrDash(ex.Status),
			ex.RepoOwner + "/" + ex.RepoName,
			roseTime(&started),
			roseTime(ex.CompletedAt),
		}
	}
	f.PrintTable(headers, rows)

	if list.Pagination.HasMore {
		f.PrintPaginationHint(list.Pagination.Total, roseExecutionsListOffset, roseExecutionsListLimit)
	}
	return nil
}
