package cmd

import (
	"encoding/json"
	"fmt"
	"strconv"

	"github.com/ollygarden/ollygarden-cli/internal/output"
	"github.com/spf13/cobra"
)

type roseRepositoryDetail struct {
	ID                   string            `json:"id"`
	RepoFullName         string            `json:"repo_full_name"`
	RepoURL              string            `json:"repo_url"`
	IsActive             bool              `json:"is_active"`
	VCSProvider          string            `json:"vcs_provider"`
	AccessStatus         string            `json:"repository_access_status"`
	LastScannedAt        *string           `json:"last_scanned_at"`
	LastScannedCommitSHA *string           `json:"last_scanned_commit_sha"`
	DashboardIssueNumber *int              `json:"dashboard_issue_number"`
	ActiveFindingsCount  int               `json:"active_findings_count"`
	FindingCounts        roseFindingCounts `json:"finding_counts"`
}

type roseInstrumentationMetadata struct {
	OTelPresent          *bool    `json:"otel_present"`
	Signals              []string `json:"signals"`
	DetectedSDKs         []string `json:"detected_sdks"`
	InstrumentationTypes []string `json:"instrumentation_types"`
	SummaryText          *string  `json:"summary_text"`
}

type roseRepositoryDetailData struct {
	Repository              roseRepositoryDetail         `json:"repository"`
	InstrumentationMetadata *roseInstrumentationMetadata `json:"instrumentation_metadata"`
	Findings                []roseRepositoryFinding      `json:"findings"`
}

var roseRepositoriesGetCmd = &cobra.Command{
	Use:   "get <repository-id>",
	Short: "Show a repository's state, instrumentation, and active findings",
	Args:  cobra.ExactArgs(1),
	RunE:  runRoseRepositoriesGet,
}

func init() {
	roseRepositoriesCmd.AddCommand(roseRepositoriesGetCmd)
}

func runRoseRepositoriesGet(cmd *cobra.Command, args []string) error {
	f := newFormatter(cmd)

	apiResp, err := roseGet(cmd.Context(), f, "/rose/repositories/"+args[0], nil)
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

	var detail roseRepositoryDetailData
	if err := json.Unmarshal(apiResp.Data, &detail); err != nil {
		return fmt.Errorf("parsing repository data: %w", err)
	}

	repo := detail.Repository
	lastScanned := roseTime(repo.LastScannedAt)
	if sha := shortSHA(repo.LastScannedCommitSHA); sha != emDash && lastScanned != emDash {
		lastScanned += " (sha " + sha + ")"
	}
	dashboardIssue := emDash
	if repo.DashboardIssueNumber != nil {
		dashboardIssue = "#" + strconv.Itoa(*repo.DashboardIssueNumber)
	}
	counts := repo.FindingCounts
	pairs := []output.KVPair{
		{Key: "ID", Value: repo.ID},
		{Key: "Repository", Value: repo.RepoFullName},
		{Key: "URL", Value: repo.RepoURL},
		{Key: "Provider", Value: orDash(repo.VCSProvider)},
		{Key: "Active", Value: boolYesNo(repo.IsActive)},
		{Key: "Access status", Value: orDash(repo.AccessStatus)},
		{Key: "Last scanned", Value: lastScanned},
		{Key: "Dashboard issue", Value: dashboardIssue},
		{Key: "Active findings", Value: fmt.Sprintf("%d (critical %d, high %d, medium %d, low %d, suggestion %d)",
			repo.ActiveFindingsCount, counts.Critical, counts.High, counts.Medium, counts.Low, counts.Suggestion)},
	}
	f.PrintKeyValue(pairs)

	out := cmd.OutOrStdout()
	if meta := detail.InstrumentationMetadata; meta != nil {
		otelPresent := emDash
		if meta.OTelPresent != nil {
			otelPresent = boolYesNo(*meta.OTelPresent)
		}
		fmt.Fprintln(out)
		fmt.Fprintln(out, "Instrumentation:")
		f.PrintKeyValue([]output.KVPair{
			{Key: "  OTel present", Value: otelPresent},
			{Key: "  Signals", Value: joinOrDash(meta.Signals)},
			{Key: "  Detected SDKs", Value: joinOrDash(meta.DetectedSDKs)},
			{Key: "  Types", Value: joinOrDash(meta.InstrumentationTypes)},
			{Key: "  Summary", Value: strOrDash(meta.SummaryText)},
		})
	}

	if len(detail.Findings) > 0 {
		fmt.Fprintln(out)
		fmt.Fprintln(out, "Findings:")
		headers := []string{"FINDING ID", "SEVERITY", "CATEGORY", "PR", "TITLE"}
		rows := make([][]string, len(detail.Findings))
		for i, fd := range detail.Findings {
			title := fd.Title
			if fd.DisplayTitle != nil && *fd.DisplayTitle != "" {
				title = *fd.DisplayTitle
			}
			rows[i] = []string{fd.FindingID, fd.Severity, strOrDash(fd.Category), prRef(fd.PRNumber, fd.PRStatus), title}
		}
		f.PrintTable(headers, rows)
	}
	return nil
}
