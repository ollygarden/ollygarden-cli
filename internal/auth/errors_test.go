package auth

import (
	"errors"
	"testing"

	"github.com/ollygarden/ollygarden-cli/internal/exitcode"
)

func TestError_CodeAndExit(t *testing.T) {
	cases := []struct {
		name     string
		err      *Error
		wantCode string
		wantExit int
	}{
		{"no creds", ErrNoCredentials(), "NO_CREDENTIALS", exitcode.Auth},
		{"invalid format", ErrInvalidTokenFormat("bad"), "INVALID_TOKEN_FORMAT", exitcode.Usage},
		{"rejected", ErrTokenRejected(), "TOKEN_REJECTED", exitcode.Auth},
		{"not found", ErrContextNotFound("internal"), "CONTEXT_NOT_FOUND", exitcode.NotFound},
		{"unreadable", ErrConfigUnreadable("/x", errors.New("oops")), "CONFIG_UNREADABLE", exitcode.Config},
		{"write failed", ErrConfigWriteFailed("/x", errors.New("oops")), "CONFIG_WRITE_FAILED", exitcode.Config},
		{"token file", ErrTokenFileNotFound("/x"), "TOKEN_FILE_NOT_FOUND", exitcode.Usage},
		{"confirm req", ErrConfirmRequired(), "CONFIRM_REQUIRED", exitcode.Usage},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if tc.err.Code != tc.wantCode {
				t.Errorf("Code: got %q, want %q", tc.err.Code, tc.wantCode)
			}
			if tc.err.ExitCode != tc.wantExit {
				t.Errorf("ExitCode: got %d, want %d", tc.err.ExitCode, tc.wantExit)
			}
			if tc.err.Message == "" {
				t.Error("Message must be non-empty")
			}
		})
	}
}

func TestError_AsTarget(t *testing.T) {
	src := ErrNoCredentials()
	wrapped := errorWrap(src)
	var got *Error
	if !errors.As(wrapped, &got) {
		t.Fatal("errors.As failed to unwrap")
	}
	if got.Code != "NO_CREDENTIALS" {
		t.Errorf("got.Code = %q", got.Code)
	}
}

// errorWrap wraps for the As-target test (avoids fmt import in the test file proper).
func errorWrap(e *Error) error {
	return &outer{e: e}
}

type outer struct{ e *Error }

func (o *outer) Error() string { return "outer: " + o.e.Error() }
func (o *outer) Unwrap() error { return o.e }
