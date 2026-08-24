package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/ollygarden/ollygarden-cli/internal/client"
	"github.com/ollygarden/ollygarden-cli/internal/output"
	"github.com/spf13/cobra"
)

// Rose endpoints are proxied verbatim by the API, so their list envelopes
// carry pagination inside `data` rather than in the top-level `meta`.
type rosePagination struct {
	Limit   int  `json:"limit"`
	Offset  int  `json:"offset"`
	Total   int  `json:"total"`
	HasMore bool `json:"hasMore"`
}

type roseListEnvelope struct {
	Data       json.RawMessage `json:"data"`
	Pagination rosePagination  `json:"pagination"`
}

const emDash = "—"

var roseUUIDPattern = regexp.MustCompile(`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$`)

// roseGet performs a GET against a Rose endpoint and parses the envelope.
// API errors are printed through the formatter (so --json mode gets the
// error envelope) before being returned for exit-code mapping.
func roseGet(ctx context.Context, f *output.Formatter, path string, query url.Values) (*client.APIResponse, error) {
	c := NewClient()
	resp, err := c.Get(ctx, path, query)
	if err != nil {
		return nil, fmt.Errorf("requesting %s: %w", path, err)
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
		return nil, err
	}
	return apiResp, nil
}

func newFormatter(cmd *cobra.Command) *output.Formatter {
	return output.New(cmd.OutOrStdout(), cmd.ErrOrStderr(), jsonMode, quiet)
}

func roseInvalidParameters(f *output.Formatter, message string) error {
	errResp := &client.ErrorResponse{
		Error: client.ErrorDetail{
			Code:    "INVALID_PARAMETERS",
			Message: message,
		},
	}
	apiErr := &client.APIError{StatusCode: http.StatusBadRequest, ErrorResponse: errResp}
	raw, _ := json.Marshal(errResp)
	f.PrintError(apiErr.Error(), raw)
	return apiErr
}

func roseCSVValuesAllowed(value string, allowed ...string) bool {
	if value == "" {
		return true
	}
	for _, item := range strings.Split(value, ",") {
		valid := false
		for _, candidate := range allowed {
			if item == candidate {
				valid = true
				break
			}
		}
		if !valid {
			return false
		}
	}
	return true
}

func orDash(s string) string {
	if s == "" {
		return emDash
	}
	return s
}

func strOrDash(s *string) string {
	if s == nil {
		return emDash
	}
	return orDash(*s)
}

func boolYesNo(b bool) string {
	if b {
		return "yes"
	}
	return "no"
}

// roseTime normalizes the mixed timestamp formats Rose emits (Postgres
// "2006-01-02 15:04:05.999999+00" and RFC 3339) to a compact UTC RFC 3339
// string. Unknown formats are returned unchanged rather than dropped.
func roseTime(s *string) string {
	if s == nil || *s == "" {
		return emDash
	}
	for _, layout := range []string{
		"2006-01-02 15:04:05.999999999-07",
		"2006-01-02 15:04:05-07",
		time.RFC3339Nano,
	} {
		if t, err := time.Parse(layout, *s); err == nil {
			return t.UTC().Format(time.RFC3339)
		}
	}
	return *s
}

func shortSHA(s *string) string {
	if s == nil || *s == "" {
		return emDash
	}
	if len(*s) > 7 {
		return (*s)[:7]
	}
	return *s
}

// prRef renders a finding's linked pull request as "#N (status)".
func prRef(number *int, status *string) string {
	if number == nil {
		return emDash
	}
	ref := "#" + strconv.Itoa(*number)
	if status != nil && *status != "" {
		ref += " (" + *status + ")"
	}
	return ref
}

func joinOrDash(parts []string) string {
	if len(parts) == 0 {
		return emDash
	}
	return strings.Join(parts, ", ")
}
