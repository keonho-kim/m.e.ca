// Purpose: Identify input-context exhaustion consistently across provider errors and local tokenization.
// Pattern: Typed model-boundary error.
// Usage: Providers preserve the cause; the run owner chooses bounded compaction recovery.
// Related: internal/agent/model/output.go, internal/provider/transport/errors.go, internal/agent/run/run.go.
package model

import "fmt"

type ContextLimitError struct {
	Input, Capacity int
	Cause           error
}

func (e *ContextLimitError) Error() string {
	if e.Cause != nil {
		return e.Cause.Error()
	}
	if e.Capacity <= 0 {
		return "model context limit exceeded; history was preserved"
	}
	return fmt.Sprintf("model context exhausted: input %d tokens, capacity %d; history was preserved", e.Input, e.Capacity)
}

func (e *ContextLimitError) Unwrap() error { return e.Cause }
