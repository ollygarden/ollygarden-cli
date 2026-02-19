package cmd

import (
	"encoding/json"
	"fmt"
	"strconv"

	"github.com/ollygarden/ollygarden-cli/internal/client"
	"github.com/ollygarden/ollygarden-cli/internal/output"
	"github.com/spf13/cobra"
)

type serviceDetail struct {
	ID                   string               `json:"id"`
	Name                 string               `json:"name"`
	Version              string               `json:"version"`
	Environment          string               `json:"environment"`
	Namespace            string               `json:"namespace"`
	FirstSeenAt          string               `json:"first_seen_at"`
	LastSeenAt           string               `json:"last_seen_at"`
	InstrumentationScore *serviceScoreCompact `json:"instrumentation_score"`
}

var servicesGetCmd = &cobra.Command{
	Use:   "get <service-id>",
	Short: "Show service details",
	Args:  cobra.ExactArgs(1),
	RunE:  runServicesGet,
}

func init() {
	servicesCmd.AddCommand(servicesGetCmd)
}

func runServicesGet(cmd *cobra.Command, args []string) error {
	serviceID := args[0]

	c := NewClient()
	f := output.New(cmd.OutOrStdout(), cmd.ErrOrStderr(), jsonMode, quiet)

	resp, err := c.Get(cmd.Context(), "/services/"+serviceID, nil)
	if err != nil {
		return fmt.Errorf("requesting service: %w", err)
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

	var svc serviceDetail
	if err := json.Unmarshal(apiResp.Data, &svc); err != nil {
		return fmt.Errorf("parsing service data: %w", err)
	}

	emDash := "\u2014"
	valOrDash := func(s string) string {
		if s == "" {
			return emDash
		}
		return s
	}

	score := emDash
	if svc.InstrumentationScore != nil {
		score = strconv.Itoa(svc.InstrumentationScore.Score)
	}

	pairs := []output.KVPair{
		{Key: "ID", Value: svc.ID},
		{Key: "Name", Value: svc.Name},
		{Key: "Version", Value: valOrDash(svc.Version)},
		{Key: "Environment", Value: valOrDash(svc.Environment)},
		{Key: "Namespace", Value: valOrDash(svc.Namespace)},
		{Key: "First Seen", Value: svc.FirstSeenAt},
		{Key: "Last Seen", Value: svc.LastSeenAt},
		{Key: "Score", Value: score},
	}

	f.PrintKeyValue(pairs)
	return nil
}
