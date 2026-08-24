package cmd

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"github.com/ollygarden/ollygarden-cli/internal/client"
	"github.com/ollygarden/ollygarden-cli/internal/output"
	"github.com/spf13/cobra"
)

type roseFindingLocation struct {
	File        string  `json:"file"`
	Line        int     `json:"line"`
	Description *string `json:"description"`
}

type roseRepositoryFinding struct {
	ID           string                `json:"id"`
	ExecutionID  string                `json:"execution_id"`
	FindingID    string                `json:"finding_id"`
	Severity     string                `json:"severity"`
	Title        string                `json:"title"`
	DisplayTitle *string               `json:"display_title"`
	Summary      *string               `json:"summary"`
	Locations    []roseFindingLocation `json:"locations"`
	Why          *string               `json:"why"`
	Fix          *string               `json:"fix"`
	Category     *string               `json:"category"`
	Checked      *bool                 `json:"checked"`
	PRNumber     *int                  `json:"pr_number"`
	PRStatus     *string               `json:"pr_status"`
	FixStatus    *string               `json:"fix_status"`
	CreatedAt    *string               `json:"created_at"`
	UpdatedAt    *string               `json:"updated_at"`
}

var roseFindingsGetCmd = &cobra.Command{
	Use:   "get <repository-id> <finding-id>",
	Short: "Show a finding's details (summary, why, fix, locations)",
	Args:  cobra.ExactArgs(2),
	RunE:  runRoseFindingsGet,
}

func init() {
	roseFindingsCmd.AddCommand(roseFindingsGetCmd)
}

// There is no per-finding endpoint; full finding detail is only embedded in
// the repository detail response, so the repository ID is required to
// locate it.
func runRoseFindingsGet(cmd *cobra.Command, args []string) error {
	repositoryID, findingID := args[0], args[1]
	f := newFormatter(cmd)

	apiResp, err := roseGet(cmd.Context(), f, "/rose/repositories/"+repositoryID, nil)
	if err != nil {
		return err
	}

	var detail roseRepositoryDetailData
	if err := json.Unmarshal(apiResp.Data, &detail); err != nil {
		return fmt.Errorf("parsing repository data: %w", err)
	}

	var finding *roseRepositoryFinding
	for i := range detail.Findings {
		if detail.Findings[i].FindingID == findingID {
			finding = &detail.Findings[i]
			break
		}
	}
	if finding == nil {
		apiErr := &client.APIError{
			StatusCode: 404,
			ErrorResponse: &client.ErrorResponse{
				Error: client.ErrorDetail{
					Code:    "FINDING_NOT_FOUND",
					Message: fmt.Sprintf("Finding %q not found in repository %s", findingID, repositoryID),
				},
				Meta: apiResp.Meta,
			},
		}
		var raw json.RawMessage
		raw, _ = json.Marshal(apiErr.ErrorResponse)
		f.PrintError(apiErr.Error(), raw)
		return apiErr
	}

	if f.IsJSON() {
		// Re-wrap the single finding in the standard envelope so --json
		// output stays {data, meta} like every other command.
		data, _ := json.Marshal(finding)
		raw, _ := json.Marshal(client.APIResponse{Data: data, Meta: apiResp.Meta})
		f.PrintJSON(raw)
		return nil
	}
	if f.IsQuiet() {
		return nil
	}

	title := finding.Title
	if finding.DisplayTitle != nil && *finding.DisplayTitle != "" {
		title = *finding.DisplayTitle
	}
	status := "active"
	if finding.Checked != nil && *finding.Checked {
		status = "resolved"
	}
	pairs := []output.KVPair{
		{Key: "Finding ID", Value: finding.FindingID},
		{Key: "Severity", Value: finding.Severity},
		{Key: "Category", Value: strOrDash(finding.Category)},
		{Key: "Title", Value: title},
		{Key: "Repository", Value: detail.Repository.RepoFullName + " (" + detail.Repository.ID + ")"},
		{Key: "Execution", Value: finding.ExecutionID},
		{Key: "Status", Value: status},
		{Key: "Fix status", Value: strOrDash(finding.FixStatus)},
		{Key: "PR", Value: prRef(finding.PRNumber, finding.PRStatus)},
		{Key: "Created", Value: roseTime(finding.CreatedAt)},
		{Key: "Updated", Value: roseTime(finding.UpdatedAt)},
	}
	f.PrintKeyValue(pairs)

	out := cmd.OutOrStdout()
	for _, section := range []struct {
		name string
		body *string
	}{
		{"Summary", finding.Summary},
		{"Why", finding.Why},
		{"Fix", finding.Fix},
	} {
		if section.body == nil || strings.TrimSpace(*section.body) == "" {
			continue
		}
		fmt.Fprintf(out, "\n%s:\n  %s\n", section.name, strings.ReplaceAll(strings.TrimSpace(*section.body), "\n", "\n  "))
	}

	if len(finding.Locations) > 0 {
		fmt.Fprintln(out)
		fmt.Fprintln(out, "Locations:")
		for _, loc := range finding.Locations {
			line := "  " + loc.File + ":" + strconv.Itoa(loc.Line)
			if loc.Description != nil && *loc.Description != "" {
				line += "  " + *loc.Description
			}
			fmt.Fprintln(out, line)
		}
	}
	return nil
}
