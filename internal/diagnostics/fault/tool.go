// Purpose: Own structured actionable failures and recovery guidance for all tool consumers.
// Pattern: Typed domain errors.
// Usage: Tool adapters return failures and the runtime preserves their code and next action.
// Related: internal/diagnostics/catalog/codes.go, internal/agent/tool/execution.go.
package fault

import (
	"errors"

	diagcatalog "github.com/keonho-kim/meca/internal/diagnostics/catalog"
)

type Failure struct {
	Code   string `json:"code"`
	Detail string `json:"detail"`
	Next   string `json:"next"`
	Cause  error  `json:"-"`
}

// NewInstructionChanged describes a rejected candidate, never a failed file mutation.
func NewInstructionChanged() *Failure {
	return &Failure{Code: diagcatalog.InstructionChanged, Detail: diagcatalog.Text(diagcatalog.ToolAUserInstructionArrivedAfterTheModel), Next: diagcatalog.Text(diagcatalog.ToolReadTheLatestUserMessagesAndReconsider)}
}

func (f *Failure) Error() string {
	return diagcatalog.Text(diagcatalog.ToolFailureDescription, f.Code, f.Detail, f.Next)
}
func (f *Failure) Unwrap() error { return f.Cause }

func (f *Failure) ErrorCode() string { return diagcatalog.Semantic(f.Code) }

func FailureOf(err error) *Failure {
	if err == nil {
		return nil
	}
	class := Code(err)
	var failure *Failure
	if errors.As(err, &failure) {
		copy := *failure
		if diagcatalog.Priority(class) > diagcatalog.Priority(failure.ErrorCode()) {
			copy.Code = class
			copy.Next = diagcatalog.Text(diagcatalog.ToolInspectState)
		}
		if err.Error() != failure.Error() {
			copy.Detail = err.Error()
		}
		return &copy
	}
	code, next := diagcatalog.ExecutionFailed, diagcatalog.Text(diagcatalog.ToolInspectState)
	switch {
	case diagcatalog.Semantic(class) == diagcatalog.DeadlineExceeded:
		code, next = diagcatalog.Timeout, diagcatalog.Text(diagcatalog.ToolInspectTimeout)
	case diagcatalog.Semantic(class) == diagcatalog.Canceled:
		code, next = diagcatalog.Interrupted, diagcatalog.Text(diagcatalog.ToolInspectInterruption)
	case diagcatalog.Semantic(class) == diagcatalog.NotFound:
		code, next = diagcatalog.PathNotFound, diagcatalog.Text(diagcatalog.ToolInspectPath)
	}
	return &Failure{Code: code, Detail: err.Error(), Next: next, Cause: err}
}
