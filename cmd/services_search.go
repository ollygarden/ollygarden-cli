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
	servicesSearchQuery       string
	servicesSearchLimit       int
	servicesSearchOffset      int
	servicesSearchEnvironment string
	servicesSearchNamespace   string
)

var servicesSearchCmd = &cobra.Command{
	Use:   "search [query]",
	Short: "Search services",
	Args:  cobra.MaximumNArgs(1),
	RunE:  runServicesSearch,
}

func init() {
	servicesCmd.AddCommand(servicesSearchCmd)
	servicesSearchCmd.Flags().StringVar(&servicesSearchQuery, "query", "", "Search query text")
	servicesSearchCmd.Flags().IntVar(&servicesSearchLimit, "limit", 20, "Maximum number of results (1-100)")
	servicesSearchCmd.Flags().IntVar(&servicesSearchOffset, "offset", 0, "Number of results to skip (≥0)")
	servicesSearchCmd.Flags().StringVar(&servicesSearchEnvironment, "environment", "", "Filter by environment")
	servicesSearchCmd.Flags().StringVar(&servicesSearchNamespace, "namespace", "", "Filter by namespace")
}

func runServicesSearch(cmd *cobra.Command, args []string) error {
	// Resolve query: positional arg takes precedence over --query flag
	q := servicesSearchQuery
	if len(args) > 0 {
		q = args[0]
	}
	if q == "" {
		return fmt.Errorf("query is required (positional arg or --query flag)")
	}

	if servicesSearchLimit < 1 || servicesSearchLimit > 100 {
		return fmt.Errorf("--limit must be between 1 and 100")
	}
	if servicesSearchOffset < 0 {
		return fmt.Errorf("--offset must be >= 0")
	}

	c := NewClient()
	f := output.New(cmd.OutOrStdout(), cmd.ErrOrStderr(), jsonMode, quiet)

	query := url.Values{}
	query.Set("q", q)
	query.Set("limit", strconv.Itoa(servicesSearchLimit))
	query.Set("offset", strconv.Itoa(servicesSearchOffset))
	if servicesSearchEnvironment != "" {
		query.Set("environment", servicesSearchEnvironment)
	}
	if servicesSearchNamespace != "" {
		query.Set("namespace", servicesSearchNamespace)
	}

	resp, err := c.Get(cmd.Context(), "/services/search", query)
	if err != nil {
		return fmt.Errorf("searching services: %w", err)
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

	var services []serviceItem
	if err := json.Unmarshal(apiResp.Data, &services); err != nil {
		return fmt.Errorf("parsing search results: %w", err)
	}

	headers := []string{"ID", "NAME", "ENVIRONMENT", "LAST SEEN", "SCORE"}
	rows := make([][]string, len(services))
	for i, s := range services {
		score := "\u2014"
		if s.InstrumentationScore != nil {
			score = strconv.Itoa(s.InstrumentationScore.Score)
		}
		rows[i] = []string{s.ID, s.Name, s.Environment, s.LastSeenAt, score}
	}

	f.PrintTable(headers, rows)

	if apiResp.Meta.HasMore {
		f.PrintPaginationHint(apiResp.Meta.Total, servicesSearchOffset, servicesSearchLimit)
	}

	return nil
}
