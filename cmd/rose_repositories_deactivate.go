package cmd

import (
	"bufio"
	"fmt"
	"strings"

	"github.com/spf13/cobra"
)

var roseRepositoriesDeactivateConfirm bool

var roseRepositoriesDeactivateCmd = &cobra.Command{
	Use:   "deactivate <repository-id>",
	Short: "Deactivate Rose analysis for a repository",
	Args:  cobra.ExactArgs(1),
	RunE:  runRoseRepositoriesDeactivate,
}

func init() {
	roseRepositoriesCmd.AddCommand(roseRepositoriesDeactivateCmd)
	roseRepositoriesDeactivateCmd.Flags().BoolVar(&roseRepositoriesDeactivateConfirm, "confirm", false, "Skip interactive confirmation")
}

func runRoseRepositoriesDeactivate(cmd *cobra.Command, args []string) error {
	f := newFormatter(cmd)
	if err := validateRoseRepositoryID(f, args[0]); err != nil {
		return err
	}
	if !roseRepositoriesDeactivateConfirm && !stdinIsTerminal() {
		return roseInvalidParameters(f, "--confirm required for non-interactive repository deactivation")
	}
	if !roseRepositoriesDeactivateConfirm {
		fmt.Fprintf(cmd.ErrOrStderr(), "Deactivate repository (id: %s)? [y/N]: ", args[0])
		line, _ := bufio.NewReader(stdinReader).ReadString('\n')
		if !strings.EqualFold(strings.TrimSpace(line), "y") {
			fmt.Fprintln(cmd.ErrOrStderr(), "Aborted.")
			return nil
		}
	}
	apiResp, err := rosePatch(cmd.Context(), f, "/rose/repositories/"+args[0], roseRepositoryActivationRequest{IsActive: false})
	if err != nil {
		return err
	}
	return printRoseRepositoryActivation(cmd, f, apiResp)
}
