package cmd

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/ollygarden/ollygarden-cli/internal/client"
	"github.com/ollygarden/ollygarden-cli/internal/output"
	"github.com/spf13/cobra"
)

type organizationData struct {
	Tier  organizationTier   `json:"tier"`
	Score *organizationScore `json:"score"`
}

type organizationTier struct {
	Name                string   `json:"name"`
	Features            []string `json:"features"`
	AllowedInsightTypes []string `json:"allowed_insight_types"`
}

type organizationScore struct {
	Value     int    `json:"value"`
	UpdatedAt string `json:"updated_at"`
}

var organizationCmd = &cobra.Command{
	Use:   "organization",
	Short: "Show organization details",
	Args:  cobra.NoArgs,
	RunE:  runOrganization,
}

func init() {
	rootCmd.AddCommand(organizationCmd)
}

func runOrganization(cmd *cobra.Command, args []string) error {
	c := NewClient()
	f := output.New(cmd.OutOrStdout(), cmd.ErrOrStderr(), jsonMode, quiet)

	resp, err := c.Get(cmd.Context(), "/organization", nil)
	if err != nil {
		return fmt.Errorf("requesting organization: %w", err)
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

	var org organizationData
	if err := json.Unmarshal(apiResp.Data, &org); err != nil {
		return fmt.Errorf("parsing organization data: %w", err)
	}

	insightTypes := "all"
	if len(org.Tier.AllowedInsightTypes) > 0 {
		insightTypes = strings.Join(org.Tier.AllowedInsightTypes, ", ")
	}

	pairs := []output.KVPair{
		{Key: "Tier", Value: org.Tier.Name},
		{Key: "Features", Value: strings.Join(org.Tier.Features, ", ")},
		{Key: "Insight Types", Value: insightTypes},
	}

	if org.Score != nil {
		pairs = append(pairs,
			output.KVPair{Key: "Score", Value: fmt.Sprintf("%d", org.Score.Value)},
			output.KVPair{Key: "Score Updated", Value: org.Score.UpdatedAt},
		)
	} else {
		pairs = append(pairs,
			output.KVPair{Key: "Score", Value: "\u2014"},
			output.KVPair{Key: "Score Updated", Value: "\u2014"},
		)
	}

	f.PrintKeyValue(pairs)
	return nil
}
