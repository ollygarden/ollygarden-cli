package client

import (
	"fmt"

	"github.com/ollygarden/ollygarden-cli/internal/exitcode"
)

// APIError represents an error returned by the OllyGarden API.
type APIError struct {
	StatusCode    int
	ErrorResponse *ErrorResponse
}

func (e *APIError) Error() string {
	if e.ErrorResponse != nil && e.ErrorResponse.Error.Message != "" {
		msg := fmt.Sprintf("Error: %s", e.ErrorResponse.Error.Message)
		if e.ErrorResponse.Meta.TraceID != "" {
			msg += fmt.Sprintf(" (trace_id: %s)", e.ErrorResponse.Meta.TraceID)
		}
		return msg
	}
	return fmt.Sprintf("Error: HTTP %d", e.StatusCode)
}

// ExitCode maps HTTP status to CLI exit code per CLI.md §5.
func (e *APIError) ExitCode() int {
	switch {
	case e.StatusCode == 400:
		return exitcode.Usage
	case e.StatusCode == 401:
		return exitcode.Auth
	case e.StatusCode == 404:
		return exitcode.NotFound
	case e.StatusCode == 429:
		return exitcode.RateLimit
	case e.StatusCode >= 500:
		return exitcode.Server
	default:
		return exitcode.General
	}
}

// ExitCodeFromError unwraps to *APIError and returns its exit code, or 1.
func ExitCodeFromError(err error) int {
	if err == nil {
		return exitcode.Success
	}
	if apiErr, ok := err.(*APIError); ok {
		return apiErr.ExitCode()
	}
	return exitcode.General
}
