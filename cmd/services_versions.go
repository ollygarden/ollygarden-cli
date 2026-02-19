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

var servicesVersionsLimit int

var servicesVersionsCmd = &cobra.Command{
	Use:   "versions <service-id>",
	Short: "List related service versions",
	Args:  cobra.ExactArgs(1),
	RunE:  runServicesVersions,
}

func init() {
	servicesCmd.AddCommand(servicesVersionsCmd)
	servicesVersionsCmd.Flags().IntVar(&servicesVersionsLimit, "limit", 20, "Maximum number of versions (1-50)")
}

func runServicesVersions(cmd *cobra.Command, args []string) error {
	if servicesVersionsLimit < 1 || servicesVersionsLimit > 50 {
		return fmt.Errorf("Error: --limit must be between 1 and 50")
	}

	serviceID := args[0]

	c := NewClient()
	f := output.New(cmd.OutOrStdout(), cmd.ErrOrStderr(), jsonMode, quiet)

	query := url.Values{}
	query.Set("limit", strconv.Itoa(servicesVersionsLimit))

	resp, err := c.Get(cmd.Context(), "/services/"+serviceID+"/versions", query)
	if err != nil {
		return fmt.Errorf("requesting service versions: %w", err)
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
		return fmt.Errorf("parsing versions data: %w", err)
	}

	headers := []string{"ID", "VERSION", "ENVIRONMENT", "LAST SEEN", "SCORE"}
	rows := make([][]string, len(services))
	for i, s := range services {
		score := "\u2014"
		if s.InstrumentationScore != nil {
			score = strconv.Itoa(s.InstrumentationScore.Score)
		}
		rows[i] = []string{s.ID, s.Version, s.Environment, s.LastSeenAt, score}
	}

	f.PrintTable(headers, rows)

	return nil
}
