// Purpose: Execute one normal agent request with text or JSONL presentation and no terminal.
// Pattern: Command use case with an output adapter.
// Usage: cmd/meca supplies the same request dependencies used by the TUI.
// Related: internal/agent/session/request.go, cmd/meca/headless.go.
package headless

import (
	"context"
	"fmt"
	sessionrun "github.com/keonho-kim/meca/internal/agent/session"
	"io"
)

func Run(ctx context.Context, request sessionrun.Request, output io.Writer, jsonMode bool) error {
	if request.Runner.Redact == nil {
		return fmt.Errorf("a secret redactor is required")
	}
	view := newOutput(output, jsonMode, request.Runner.Redact)
	if err := view.begin(request.Runner.Session, request.Runner.ProjectRoot); err != nil {
		return err
	}
	result, err := sessionrun.Execute(ctx, request, noApproval, view.emit)
	return view.finish(result.Status, result.Text, err)
}
