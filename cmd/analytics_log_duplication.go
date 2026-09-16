package cmd

import (
	"encoding/json"
	"fmt"

	"github.com/ollygarden/ollygarden-cli/internal/output"
	"github.com/spf13/cobra"
)

type logDuplication struct {
	Total             int64   `json:"total"`
	UniqueLogMessages int64   `json:"unique_log_messages"`
	DedupPotentialPct float64 `json:"dedup_potential_pct"`
}

var analyticsLogDuplicationCmd = &cobra.Command{
	Use:   "log-duplication",
	Short: "Total versus distinct log bodies and the dedup potential",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runMagnoliaWidget(cmd, "/magnolia/log-duplication", renderLogDuplication)
	},
}

func renderLogDuplication(f *output.Formatter, data json.RawMessage) error {
	var result logDuplication
	if err := json.Unmarshal(data, &result); err != nil {
		return fmt.Errorf("parsing log duplication: %w", err)
	}
	f.PrintKeyValue([]output.KVPair{
		{Key: "Total log records", Value: fmt.Sprint(result.Total)},
		{Key: "Unique messages", Value: fmt.Sprint(result.UniqueLogMessages)},
		{Key: "Dedup potential", Value: fmt.Sprintf("%.1f%%", result.DedupPotentialPct)},
	})
	return nil
}

func init() {
	analyticsCmd.AddCommand(analyticsLogDuplicationCmd)
}
