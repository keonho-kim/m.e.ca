// Purpose: Define read-only file browsing independently of agent tools and terminal state.
// Pattern: Consumer-owned filesystem I/O boundary.
// Usage: Local tools lend their guarded roots; the TUI consumes Source.
// Related: internal/fileview/directory.go, internal/fileview/preview.go, internal/tools/local/executor.go.
package fileview

import (
	"context"
	"os"

	"github.com/keonho-kim/meca/internal/policy"
)

type Target struct {
	Absolute, Relative string
	Root               *os.Root
}

type Source interface {
	Reference(context.Context, policy.Settings, string) (Reference, error)
	Directory(context.Context, policy.Settings, string, int, int) ([]Entry, bool, error)
	Preview(context.Context, policy.Settings, string) (Preview, error)
}

type Reader struct {
	project string
	resolve func(policy.Scope, string) (Target, error)
	redact  func(string) string
}

// New borrows guarded roots through resolve; their owner closes them after readers stop.
func New(project string, resolve func(policy.Scope, string) (Target, error), redact func(string) string) *Reader {
	return &Reader{project: project, resolve: resolve, redact: redact}
}
