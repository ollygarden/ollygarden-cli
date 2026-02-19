package cmd

import (
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"

	"github.com/ollygarden/ollygarden-cli/internal/client"
	"github.com/ollygarden/ollygarden-cli/internal/output"
	"github.com/spf13/cobra"
)

var (
	servicesGroupedLimit  int
	servicesGroupedOffset int
	servicesGroupedSort   string
)

var allowedSorts = map[string]bool{
	"insights-first": true,
	"name-asc":       true,
	"name-desc":      true,
	"created-asc":    true,
	"created-desc":   true,
}

type groupedServiceItem struct {
	Name                 string               `json:"name"`
	Environment          string               `json:"environment"`
	Namespace            string               `json:"namespace"`
	LatestID             string               `json:"latest_id"`
	VersionCount         int                  `json:"version_count"`
	InsightsCount        int                  `json:"insights_count"`
	InstrumentationScore *serviceScoreCompact `json:"instrumentation_score"`
}

var servicesGroupedCmd = &cobra.Command{
	Use:   "grouped",
	Short: "List services grouped by name",
	Args:  cobra.NoArgs,
	RunE:  runServicesGrouped,
}

func init() {
	servicesCmd.AddCommand(servicesGroupedCmd)
	servicesGroupedCmd.Flags().IntVar(&servicesGroupedLimit, "limit", 50, "Maximum number of results (1-100)")
	servicesGroupedCmd.Flags().IntVar(&servicesGroupedOffset, "offset", 0, "Number of results to skip (≥0)")
	servicesGroupedCmd.Flags().StringVar(&servicesGroupedSort, "sort", "insights-first", "Sort order: insights-first, name-asc, name-desc, created-asc, created-desc")
}

func runServicesGrouped(cmd *cobra.Command, args []string) error {
	if servicesGroupedLimit < 1 || servicesGroupedLimit > 100 {
		return fmt.Errorf("Error: --limit must be between 1 and 100")
	}
	if servicesGroupedOffset < 0 {
		return fmt.Errorf("Error: --offset must be >= 0")
	}
	if !allowedSorts[servicesGroupedSort] {
		return fmt.Errorf("Error: --sort must be one of: insights-first, name-asc, name-desc, created-asc, created-desc")
	}

	c := NewClient()
	f := output.New(cmd.OutOrStdout(), cmd.ErrOrStderr(), jsonMode, quiet)

	query := url.Values{}
	query.Set("limit", strconv.Itoa(servicesGroupedLimit))
	query.Set("offset", strconv.Itoa(servicesGroupedOffset))
	query.Set("sort", servicesGroupedSort)

	resp, err := c.Get(cmd.Context(), "/services/grouped", query)
	if err != nil {
		return fmt.Errorf("requesting grouped services: %w", err)
	}

	apiResp, err := client.ParseResponse(resp)
	if err != nil {
		if apiErr, ok := err.(*client.APIError); ok {
			var raw json.RawMessage
			if apiErr.ErrorResponse != nil {
				raw, _ = json.Marshal(apiErr.ErrorResponse)
			}
			f.PrintError(apiErr.Error(), raw)
		}
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

	var services []groupedServiceItem
	if err := json.Unmarshal(apiResp.Data, &services); err != nil {
		return fmt.Errorf("parsing grouped services data: %w", err)
	}

	headers := []string{"NAME", "ENVIRONMENT", "VERSIONS", "INSIGHTS", "SCORE"}
	rows := make([][]string, len(services))
	for i, s := range services {
		score := "\u2014"
		if s.InstrumentationScore != nil {
			score = strconv.Itoa(s.InstrumentationScore.Score)
		}
		rows[i] = []string{s.Name, s.Environment, strconv.Itoa(s.VersionCount), strconv.Itoa(s.InsightsCount), score}
	}

	f.PrintTable(headers, rows)

	if apiResp.Meta.HasMore {
		f.PrintPaginationHint(apiResp.Meta.Total, servicesGroupedOffset, servicesGroupedLimit)
	}

	return nil
}
