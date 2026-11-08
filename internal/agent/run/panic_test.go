// Purpose: Preserve uncertain tool effects and private panic handling at the execution boundary.
// Pattern: Tool failure contract test.
// Usage: go test ./internal/agent/run -run TestToolPanic.
// Related: internal/agent/run/execute.go, internal/diagnostics/fault/panic.go.
package run

import (
	"context"
	toolapi "github.com/keonho-kim/meca/internal/agent/tool"
	"github.com/keonho-kim/meca/internal/diagnostics/fault"
	"strings"
	"testing"
)

type panicOperation struct{}

func (panicOperation) Approval() toolapi.Approval { return toolapi.Approval{} }
func (panicOperation) Execute(context.Context, bool) (toolapi.Result, error) {
	panic("private-panic-value")
}

func TestToolPanicPreservesUncertainExecution(t *testing.T) {
	value, err := executePrepared(t.Context(), &readyExecution{operation: panicOperation{}})
	if !fault.IsPanic(err) || !value.Started || !value.Failed || value.Failure == nil {
		t.Fatal(value, err)
	}
	if strings.Contains(err.Error(), "private-panic-value") || strings.Contains(value.Failure.Error(), "private-panic-value") {
		t.Fatal("panic content escaped the failure boundary")
	}
}
