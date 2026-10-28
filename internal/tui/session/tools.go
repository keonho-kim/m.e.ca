// Purpose: Define agent execution capabilities consumed by the terminal session.
// Pattern: Consumer-owned I/O contract.
// Usage: The composition root supplies local.Executor independently of the fileview.Source reader.
// Related: internal/tui/session/state.go, internal/tools/local/executor.go, internal/tui/session/tasks.go.
package session

import (
	"context"

	toolapi "github.com/keonho-kim/meca/internal/agent/tool"
)

type Tools interface {
	toolapi.Executor
	ProjectInstructions(context.Context) (string, error)
}
