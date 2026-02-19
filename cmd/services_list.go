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
	servicesListLimit  int
	servicesListOffset int
)

type serviceItem struct {
	ID                   string               `json:"id"`
	Name                 string               `json:"name"`
	Environment          string               `json:"environment"`
	LastSeenAt           string               `json:"last_seen_at"`
	InstrumentationScore *serviceScoreCompact `json:"instrumentation_score"`
}

type serviceScoreCompact struct {
	Score int `json:"score"`
}

var servicesListCmd = &cobra.Command{
	Use:   "list",
	Short: "List services",
	Args:  cobra.NoArgs,
	RunE:  runServicesList,
}

func init() {
	servicesCmd.AddCommand(servicesListCmd)
	servicesListCmd.Flags().IntVar(&servicesListLimit, "limit", 50, "Maximum number of results (1-100)")
	servicesListCmd.Flags().IntVar(&servicesListOffset, "offset", 0, "Number of results to skip (≥0)")
}

func runServicesList(cmd *cobra.Command, args []string) error {
	if servicesListLimit < 1 || servicesListLimit > 100 {
		return fmt.Errorf("Error: --limit must be between 1 and 100")
	}
	if servicesListOffset < 0 {
		return fmt.Errorf("Error: --offset must be >= 0")
	}

	c := NewClient()
	f := output.New(cmd.OutOrStdout(), cmd.ErrOrStderr(), jsonMode, quiet)

	query := url.Values{}
	query.Set("limit", strconv.Itoa(servicesListLimit))
	query.Set("offset", strconv.Itoa(servicesListOffset))

	resp, err := c.Get(cmd.Context(), "/services", query)
	if err != nil {
		return fmt.Errorf("requesting services: %w", err)
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
		return fmt.Errorf("parsing services data: %w", err)
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
		f.PrintPaginationHint(apiResp.Meta.Total, servicesListOffset, servicesListLimit)
	}

	return nil
}
