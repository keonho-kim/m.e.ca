// Purpose: Define the file and tool operations consumed by the terminal session.
// Pattern: Consumer-owned I/O contract.
// Usage: The composition root supplies local.Executor; controlled readers substitute it in tests.
// Related: internal/tui/session/state.go, internal/tools/local/executor.go, internal/tui/session/directory.go.
package session

import (
	"context"

	toolapi "github.com/keonho-kim/meca/internal/agent/tool"
	"github.com/keonho-kim/meca/internal/policy"
	"github.com/keonho-kim/meca/internal/tools/local"
)

type Tools interface {
	toolapi.Executor
	ProjectInstructions(context.Context) (string, error)
	Reference(policy.Settings, string) (local.Reference, error)
	Directory(context.Context, policy.Settings, string, int, int) ([]local.Entry, bool, error)
	Preview(context.Context, policy.Settings, string) (local.FilePreview, error)
}
