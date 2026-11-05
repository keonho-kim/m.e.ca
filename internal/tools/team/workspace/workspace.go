// Purpose: Reserve overlapping mutation targets while allowing independent commands and file work.
// Pattern: Context-cancellable resource admission.
// Usage: Every cooperative local executor shares one Access for the request.
// Related: internal/tools/team/executor.go, internal/tools/team/local.go.
package workspace

import (
	"context"
	"os"
	"path/filepath"
	"sync"

	toolapi "github.com/keonho-kim/meca/internal/agent/tool"
	"github.com/keonho-kim/meca/internal/tools/local/filesystem"
)

type target struct {
	path string
	info os.FileInfo
}
type Access struct {
	mu      sync.Mutex
	waiters []*waiter
	files   int
	writes  map[*Lease][]target
}
type Lease struct {
	once      sync.Once
	workspace *Access
	command   bool
}

func New() *Access {
	return &Access{writes: make(map[*Lease][]target)}
}

func canonical(path string) target {
	path = filepath.Clean(path)
	if resolved, err := filesystem.Canonical(path); err == nil {
		path = resolved
	}
	info, _ := os.Stat(path)
	return target{path, info}
}
func overlap(a, b target) bool {
	return containsPath(a.path, b.path) || containsPath(b.path, a.path) || a.info != nil && b.info != nil && os.SameFile(a.info, b.info)
}

func containsPath(parent, child string) bool {
	return parent == child || len(child) > len(parent) && child[:len(parent)] == parent && (parent == string(filepath.Separator) || child[len(parent)] == filepath.Separator)
}

func conflicts(left, right []target) bool {
	for _, a := range left {
		for _, b := range right {
			if overlap(a, b) {
				return true
			}
		}
	}
	return false
}

func admissionError(ctx context.Context, obsolete <-chan struct{}) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	select {
	case <-obsolete:
		return toolapi.NewInstructionChanged()
	default:
		return nil
	}
}
func (l *Lease) Release() {
	if l == nil {
		return
	}
	l.once.Do(func() {
		w := l.workspace
		w.mu.Lock()
		defer w.mu.Unlock()
		keys := w.writes[l]
		delete(w.writes, l)
		if !l.command {
			w.files--
		}
		w.notify(keys, !l.command)
	})
}

// ReserveFiles keeps overlapping targets exclusive until Release.
func (w *Access) ReserveFiles(ctx context.Context, paths []string, obsolete <-chan struct{}, waiting func() error) (*Lease, error) {
	return w.reserve(ctx, paths, false, obsolete, waiting)
}

// ReserveCommand protects only known mutation paths; the owning process registry bounds commands.
func (w *Access) ReserveCommand(ctx context.Context, paths []string, obsolete <-chan struct{}, waiting func() error) (*Lease, error) {
	if len(paths) == 0 {
		return nil, admissionError(ctx, obsolete)
	}
	return w.reserve(ctx, paths, true, obsolete, waiting)
}
