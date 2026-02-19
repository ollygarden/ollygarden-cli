package cmd

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"

	"github.com/ollygarden/ollygarden-cli/internal/client"
	"github.com/ollygarden/ollygarden-cli/internal/output"
	"github.com/spf13/cobra"
	"golang.org/x/term"
)

var webhooksDeleteConfirm bool

// Testability hooks for TTY detection and stdin reading.
var (
	stdinIsTerminal = func() bool { return term.IsTerminal(int(os.Stdin.Fd())) }
	stdinReader     io.Reader = os.Stdin
)

var webhooksDeleteCmd = &cobra.Command{
	Use:   "delete <webhook-id>",
	Short: "Delete a webhook",
	Args:  cobra.ExactArgs(1),
	RunE:  runWebhooksDelete,
}

func init() {
	webhooksCmd.AddCommand(webhooksDeleteCmd)
	webhooksDeleteCmd.Flags().BoolVar(&webhooksDeleteConfirm, "confirm", false, "Skip interactive confirmation")
}

func runWebhooksDelete(cmd *cobra.Command, args []string) error {
	webhookID := args[0]
	f := output.New(cmd.OutOrStdout(), cmd.ErrOrStderr(), jsonMode, quiet)
	c := NewClient()

	// Non-TTY without --confirm → exit 2
	if !webhooksDeleteConfirm && !stdinIsTerminal() {
		return fmt.Errorf("Error: --confirm required for non-interactive webhook deletion")
	}

	// GET webhook to obtain name for confirmation prompt / success message
	getResp, err := c.Get(cmd.Context(), "/webhooks/"+webhookID, nil)
	if err != nil {
		return fmt.Errorf("fetching webhook: %w", err)
	}

	apiResp, err := client.ParseResponse(getResp)
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

	var wh webhookDetail
	if err := json.Unmarshal(apiResp.Data, &wh); err != nil {
		return fmt.Errorf("parsing webhook data: %w", err)
	}

	// Interactive confirmation prompt (TTY, no --confirm)
	if !webhooksDeleteConfirm {
		fmt.Fprintf(cmd.ErrOrStderr(), "Delete webhook %q (id: %s)? [y/N]: ", wh.Name, wh.ID)
		reader := bufio.NewReader(stdinReader)
		line, _ := reader.ReadString('\n')
		line = strings.TrimSpace(line)
		if !strings.EqualFold(line, "y") {
			fmt.Fprintln(cmd.ErrOrStderr(), "Aborted.")
			return nil
		}
	}

	// DELETE the webhook
	delResp, err := c.Delete(cmd.Context(), "/webhooks/"+webhookID)
	if err != nil {
		return fmt.Errorf("deleting webhook: %w", err)
	}
	defer delResp.Body.Close()

	// 204 No Content = success
	if delResp.StatusCode == http.StatusNoContent {
		if f.IsJSON() || f.IsQuiet() {
			return nil
		}
		fmt.Fprintf(cmd.ErrOrStderr(), "Deleted webhook %q (id: %s).\n", wh.Name, wh.ID)
		return nil
	}

	// Error response — parse error body
	body, err := io.ReadAll(delResp.Body)
	if err != nil {
		return fmt.Errorf("reading error response: %w", err)
	}

	var errResp client.ErrorResponse
	if jsonErr := json.Unmarshal(body, &errResp); jsonErr != nil {
		return &client.APIError{StatusCode: delResp.StatusCode}
	}

	apiErr := &client.APIError{StatusCode: delResp.StatusCode, ErrorResponse: &errResp}
	raw, _ := json.Marshal(&errResp)
	f.PrintError(apiErr.Error(), raw)
	return apiErr
}
