// Purpose: Coordinate shared file mutations and hold exclusive shell ownership through actual completion.
// Pattern: Context-cancellable resource admission.
// Usage: Every cooperative local executor shares one Access for the request.
// Related: internal/tools/team/executor.go, internal/tools/team/local.go.
package workspace

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/keonho-kim/meca/internal/tools/local/filesystem"
)

type target struct {
	path string
	info os.FileInfo
}
type Access struct {
	mu      sync.Mutex
	changed chan struct{}
	shell   bool
	denied  map[string]bool
	writes  map[*Lease][]target
	images  chan struct{}
}
type Lease struct {
	once      sync.Once
	workspace *Access
	shell     bool
}

func New() *Access {
	return &Access{changed: make(chan struct{}), writes: map[*Lease][]target{}, images: make(chan struct{}, 1)}
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
	return a.path == b.path || strings.HasPrefix(a.path, b.path+string(filepath.Separator)) || strings.HasPrefix(b.path, a.path+string(filepath.Separator)) || a.info != nil && b.info != nil && os.SameFile(a.info, b.info)
}
func (w *Access) reserve(ctx context.Context, paths []string, shell bool, waiting func() error) (*Lease, error) {
	announced := false
	for {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		keys := make([]target, len(paths))
		for i, path := range paths {
			keys[i] = canonical(path)
		}
		w.mu.Lock()
		ready := !w.shell
		if shell {
			ready = ready && len(w.writes) == 0
		} else {
			ready = ready && len(w.writes) < 16
			for _, held := range w.writes {
				for _, a := range held {
					for _, b := range keys {
						if overlap(a, b) {
							ready = false
						}
					}
				}
			}
		}
		if ready {
			value := &Lease{workspace: w, shell: shell}
			if shell {
				w.shell = true
			} else {
				w.writes[value] = keys
			}
			w.mu.Unlock()
			return value, nil
		}
		changed := w.changed
		w.mu.Unlock()
		if !announced && waiting != nil {
			if err := waiting(); err != nil {
				return nil, err
			}
			announced = true
		}
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-changed:
		}
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
		if l.shell {
			w.shell = false
		} else {
			delete(w.writes, l)
		}
		close(w.changed)
		w.changed = make(chan struct{})
	})
}

// ReserveFiles keeps overlapping targets exclusive until Release.
func (w *Access) ReserveFiles(ctx context.Context, paths []string, waiting func() error) (*Lease, error) {
	return w.reserve(ctx, paths, false, waiting)
}

// ReserveShell excludes file mutations and other shells through actual command completion.
func (w *Access) ReserveShell(ctx context.Context, waiting func() error) (*Lease, error) {
	return w.reserve(ctx, nil, true, waiting)
}

// AcquireImage bounds active decoding; the successful caller owns ReleaseImage.
func (w *Access) AcquireImage(ctx context.Context) error {
	select {
	case w.images <- struct{}{}:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}
func (w *Access) ReleaseImage() { <-w.images }
