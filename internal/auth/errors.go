// Package auth orchestrates login (HTTP probe + persist) and credential
// resolution (env/flag/file precedence). It depends on internal/config for
// on-disk schema and internal/client for the HTTP probe; everything else
// (formatting, prompts, exit-code routing) belongs in cmd/.
package auth

import (
	"fmt"

	"github.com/ollygarden/ollygarden-cli/internal/exitcode"
)

// Error is the typed error this package returns. Each instance carries
// a stable Code (machine-readable, mirrored in the JSON error envelope)
// and an ExitCode used by cmd.Execute() to set the process exit status.
//
// Cmd code switches on this type via errors.As. Never compare Code values
// from outside this package — use the constructors below.
type Error struct {
	Code     string
	ExitCode int
	Message  string
	Cause    error
}

func (e *Error) Error() string {
	if e.Cause == nil {
		return e.Message
	}
	return fmt.Sprintf("%s: %v", e.Message, e.Cause)
}

func (e *Error) Unwrap() error { return e.Cause }

// Constructors. One per error code from the spec.

func ErrNoCredentials() *Error {
	return &Error{
		Code:     "NO_CREDENTIALS",
		ExitCode: exitcode.Auth,
		Message:  `No credentials configured. Run "ollygarden auth login" or set OLLYGARDEN_API_KEY. Get a token at https://ollygarden.app/settings.`,
	}
}

func ErrInvalidTokenFormat(_ string) *Error {
	return &Error{
		Code:     "INVALID_TOKEN_FORMAT",
		ExitCode: exitcode.Usage,
		Message:  "Invalid token format. Expected og_sk_xxxxxx_<32 hex>.",
	}
}

func ErrTokenRejected() *Error {
	return &Error{
		Code:     "TOKEN_REJECTED",
		ExitCode: exitcode.Auth,
		Message:  "Token rejected by API. The token may be revoked or expired.",
	}
}

func ErrContextNotFound(name string) *Error {
	return &Error{
		Code:     "CONTEXT_NOT_FOUND",
		ExitCode: exitcode.NotFound,
		Message:  fmt.Sprintf(`Context %q not found. Run "ollygarden auth list-contexts" to see available contexts.`, name),
	}
}

func ErrConfigUnreadable(path string, cause error) *Error {
	return &Error{
		Code:     "CONFIG_UNREADABLE",
		ExitCode: exitcode.Config,
		Message:  fmt.Sprintf("Cannot read config file at %s. Inspect or remove the file to recover.", path),
		Cause:    cause,
	}
}

func ErrConfigWriteFailed(path string, cause error) *Error {
	return &Error{
		Code:     "CONFIG_WRITE_FAILED",
		ExitCode: exitcode.Config,
		Message:  fmt.Sprintf("Cannot write config file at %s.", path),
		Cause:    cause,
	}
}

func ErrTokenFileNotFound(path string) *Error {
	return &Error{
		Code:     "TOKEN_FILE_NOT_FOUND",
		ExitCode: exitcode.Usage,
		Message:  fmt.Sprintf("Cannot read token file %s.", path),
	}
}

func ErrConfirmRequired() *Error {
	return &Error{
		Code:     "CONFIRM_REQUIRED",
		ExitCode: exitcode.Usage,
		Message:  "Refusing to remove all contexts without --confirm in non-interactive mode.",
	}
}
