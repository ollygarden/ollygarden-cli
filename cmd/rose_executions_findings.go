package cmd

import (
	"encoding/json"
	"net/url"
	"strconv"

	"github.com/spf13/cobra"
)

var (
	roseExecutionsFindingsStatus    string
	roseExecutionsFindingsDismissed string
	roseExecutionsFindingsPage      int
	roseExecutionsFindingsLimit     int
)

var roseExecutionsFindingsCmd = &cobra.Command{
	Use:   "findings <execution-id>",
	Short: "List findings produced by a Rose execution",
	Args:  cobra.ExactArgs(1),
	RunE:  runRoseExecutionsFindings,
}

func init() {
	roseExecutionsCmd.AddCommand(roseExecutionsFindingsCmd)
	roseExecutionsFindingsCmd.Flags().StringVar(&roseExecutionsFindingsStatus, "status", "all", "Finding status (active, resolved, all)")
	roseExecutionsFindingsCmd.Flags().StringVar(&roseExecutionsFindingsDismissed, "dismissed", "false", "Dismissed filter (false, true, all)")
	roseExecutionsFindingsCmd.Flags().IntVar(&roseExecutionsFindingsPage, "page", 1, "Page number (≥1)")
	roseExecutionsFindingsCmd.Flags().IntVar(&roseExecutionsFindingsLimit, "limit", 50, "Results per page (1-100)")
}

func runRoseExecutionsFindings(cmd *cobra.Command, args []string) error {
	f := newFormatter(cmd)
	if !roseUUIDPattern.MatchString(args[0]) {
		return roseInvalidParameters(f, "execution-id must be a UUID")
	}
	if roseExecutionsFindingsLimit < 1 || roseExecutionsFindingsLimit > 100 {
		return roseInvalidParameters(f, "--limit must be between 1 and 100")
	}
	if roseExecutionsFindingsPage < 1 {
		return roseInvalidParameters(f, "--page must be >= 1")
	}
	switch roseExecutionsFindingsStatus {
	case "active", "resolved", "all":
	default:
		return roseInvalidParameters(f, "--status must be one of: active, resolved, all")
	}
	switch roseExecutionsFindingsDismissed {
	case "false", "true", "all":
	default:
		return roseInvalidParameters(f, "--dismissed must be one of: false, true, all")
	}

	query := url.Values{}
	query.Set("executionId", args[0])
	query.Set("status", roseExecutionsFindingsStatus)
	query.Set("dismissed", roseExecutionsFindingsDismissed)
	query.Set("page", strconv.Itoa(roseExecutionsFindingsPage))
	query.Set("page_size", strconv.Itoa(roseExecutionsFindingsLimit))
	apiResp, err := roseGet(cmd.Context(), f, "/rose/codebase/findings", query)
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

	list, err := printRoseFindingsTable(f, apiResp.Data)
	if err != nil {
		return err
	}
	if list.Pagination.HasMore {
		f.PrintPageHint(list.Pagination.Total, roseExecutionsFindingsPage, roseExecutionsFindingsLimit)
	}
	return nil
}
