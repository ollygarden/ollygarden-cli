package cmd

import (
	"encoding/json"
	"runtime"

	"github.com/ollygarden/ollygarden-cli/internal/output"
	"github.com/spf13/cobra"
)

type versionInfo struct {
	Version string `json:"version"`
	Commit  string `json:"commit"`
	Date    string `json:"date"`
	Go      string `json:"go"`
	OS      string `json:"os"`
	Arch    string `json:"arch"`
}

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Show CLI version and build info",
	Args:  cobra.NoArgs,
	RunE:  runVersion,
}

func init() {
	rootCmd.AddCommand(versionCmd)
}

func runVersion(cmd *cobra.Command, args []string) error {
	f := output.New(cmd.OutOrStdout(), cmd.ErrOrStderr(), jsonMode, quiet)

	info := versionInfo{
		Version: version,
		Commit:  commit,
		Date:    date,
		Go:      runtime.Version(),
		OS:      runtime.GOOS,
		Arch:    runtime.GOARCH,
	}

	if f.IsJSON() {
		raw, _ := json.Marshal(info)
		f.PrintJSON(raw)
		return nil
	}

	if f.IsQuiet() {
		return nil
	}

	f.PrintKeyValue([]output.KVPair{
		{Key: "Version", Value: info.Version},
		{Key: "Commit", Value: info.Commit},
		{Key: "Built", Value: info.Date},
		{Key: "Go", Value: info.Go},
		{Key: "OS/Arch", Value: info.OS + "/" + info.Arch},
	})
	return nil
}
