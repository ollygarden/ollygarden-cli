package cmd

import "github.com/spf13/cobra"

var authCmd = &cobra.Command{
	Use:   "auth",
	Short: "Manage OllyGarden CLI credentials",
	Long: `Manage credentials stored on disk for the OllyGarden CLI.

Credentials are kept in a YAML file at:
  os.UserConfigDir()/ollygarden/config.yaml  (mode 0600)

Override with the OLLYGARDEN_CONFIG environment variable.

The OLLYGARDEN_API_KEY environment variable always wins over saved
credentials, so CI runs and ad-hoc invocations continue to work.`,
}

func init() {
	rootCmd.AddCommand(authCmd)
}
