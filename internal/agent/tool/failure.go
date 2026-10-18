// Purpose: Describe actionable tool failures without collapsing distinct recovery paths.
// Pattern: Typed domain errors.
// Usage: Tool adapters return failures and the runtime preserves their code and next action.
// Related: internal/tools/local/prepare.go, internal/agent/run/action.go.
package tool

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
)

const (
	StaleTarget      = "file_changed"
	InvalidArgument  = "invalid_argument"
	OutputTruncated  = "output_truncated"
	Timeout          = "timeout"
	Interrupted      = "interrupted"
	ProcessFailed    = "process_failed"
	ScopeUnavailable = "scope_unavailable"
	ExecutionFailed  = "execution_failed"
)

type Failure struct {
	Code   string `json:"code"`
	Detail string `json:"detail"`
	Next   string `json:"next"`
	Cause  error  `json:"-"`
}

func (f *Failure) Error() string { return fmt.Sprintf("%s: %s; next: %s", f.Code, f.Detail, f.Next) }
func (f *Failure) Unwrap() error { return f.Cause }

func FailureOf(err error) *Failure {
	if err == nil {
		return nil
	}
	var failure *Failure
	if errors.As(err, &failure) {
		copy := *failure
		if err.Error() != failure.Error() {
			copy.Detail = err.Error()
		}
		return &copy
	}
	code, next := ExecutionFailed, "Inspect current state and choose a narrower or different approach; do not blindly replay side effects."
	switch {
	case errors.Is(err, context.DeadlineExceeded):
		code, next = Timeout, "Inspect partial output, process state and the remaining task budget before resuming; do not blindly replay side effects."
	case errors.Is(err, context.Canceled):
		code, next = Interrupted, "Inspect saved state before resuming; execution may have started."
	case errors.Is(err, fs.ErrNotExist):
		code, next = "path_not_found", "List the parent directory and verify the literal path."
	}
	return &Failure{Code: code, Detail: err.Error(), Next: next, Cause: err}
}

type ProcessOutcome struct {
	Diagnostic string
	Truncated  bool
	Status     string
	ExitCode   int
}
