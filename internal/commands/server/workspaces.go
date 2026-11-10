// Purpose: Register projects and lend lazily opened project runtimes without changing running roots.
// Pattern: Explicit workspace resource registry.
// Usage: The process composition root owns this registry; API commands borrow one project.
// Related: cmd/meca/server.go, internal/commands/server/workspaces_lifecycle.go.
package server

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"sync"
	"time"

	diag "github.com/keonho-kim/meca/internal/diagnostics/catalog"
	"github.com/keonho-kim/meca/internal/diagnostics/fault"
	"github.com/keonho-kim/meca/internal/tools/local/filesystem"
)

type Workspace struct {
	Approvals             int
	ID, Path, Name, Error string
	Active                int
}
type WorkspaceRuntime struct {
	Service *Service
	Close   func() error
}
type WorkspaceOptions struct {
	Default string
	Paths   []string
	Check   func(string) (string, error)
	Save    func(context.Context, string) error
	Open    func(context.Context, Workspace) (WorkspaceRuntime, error)
}
type workspaceEntry struct {
	value   Workspace
	runtime WorkspaceRuntime
	ready   chan struct{}
	refs    int
	used    time.Time
}
type Workspaces struct {
	mu        sync.Mutex
	ctx       context.Context
	cancel    context.CancelFunc
	options   WorkspaceOptions
	entries   map[string]*workspaceEntry
	order     []string
	defaultID string
	closed    bool
	closeDone chan struct{}
	workers   sync.WaitGroup
	closeErr  error
}

func NewWorkspaces(ctx context.Context, o WorkspaceOptions) (*Workspaces, error) {
	if !filepath.IsAbs(o.Default) || o.Check == nil || o.Save == nil || o.Open == nil {
		return nil, fault.New(diag.InvalidArgument, diag.UIWorkspaceInvalid)
	}
	ctx, cancel := context.WithCancel(ctx)
	w := &Workspaces{ctx: ctx, cancel: cancel, options: o, entries: make(map[string]*workspaceEntry), closeDone: make(chan struct{})}
	persistedDefault := false
	for index, path := range append([]string{o.Default}, o.Paths...) {
		if !filepath.IsAbs(path) {
			cancel()
			return nil, fault.New(diag.InvalidArgument, diag.UIWorkspaceInvalid)
		}
		absolute, err := filesystem.Canonical(path)
		if err != nil {
			absolute = filepath.Clean(path)
		}
		id := "workspace_" + ProjectScope(absolute)
		if index > 0 && id == w.defaultID {
			persistedDefault = true
		}
		if w.entries[id] != nil {
			continue
		}
		w.entries[id] = &workspaceEntry{value: Workspace{ID: id, Path: absolute, Name: filepath.Base(absolute)}}
		w.order = append(w.order, id)
		if w.defaultID == "" {
			w.defaultID = id
		}
	}
	if !persistedDefault {
		if err := o.Save(ctx, w.entries[w.defaultID].value.Path); err != nil {
			cancel()
			return nil, err
		}
	}
	w.workers.Add(1)
	go w.reap()
	return w, nil
}

func (w *Workspaces) List() ([]Workspace, string) {
	w.mu.Lock()
	defer w.mu.Unlock()
	values := make([]Workspace, 0, len(w.order))
	for _, id := range w.order {
		e := w.entries[id]
		value := e.value
		if e.runtime.Service != nil {
			value.Active, value.Approvals = e.runtime.Service.workspaceActivity()
		}
		values = append(values, value)
	}
	return values, w.defaultID
}

func (w *Workspaces) Add(ctx context.Context, path string) (Workspace, error) {
	if path == "" {
		return Workspace{}, fault.New(diag.InvalidArgument, diag.UIWorkspaceInvalid)
	}
	path, err := w.options.Check(path)
	if err != nil {
		return Workspace{}, err
	}
	info, err := os.Stat(path)
	if err != nil {
		return Workspace{}, err
	}
	if !info.IsDir() {
		return Workspace{}, fault.New(diag.InvalidArgument, diag.UIWorkspaceInvalid)
	}
	value := Workspace{ID: "workspace_" + ProjectScope(path), Path: path, Name: filepath.Base(path)}
	w.mu.Lock()
	if w.closed {
		w.mu.Unlock()
		return Workspace{}, fault.New(diag.Unavailable, diag.UIWorkspaceClosed)
	}
	_, exists := w.entries[value.ID]
	w.mu.Unlock()
	if exists {
		return value, nil
	}
	if err := w.options.Save(ctx, path); err != nil {
		return Workspace{}, err
	}
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.closed {
		return Workspace{}, fault.New(diag.Unavailable, diag.UIWorkspaceClosed)
	}
	if w.entries[value.ID] == nil {
		w.entries[value.ID] = &workspaceEntry{value: value}
		w.order = append(w.order, value.ID)
	}
	return value, nil
}

func (w *Workspaces) Acquire(ctx context.Context, id string) (*Service, func(), error) {
	for {
		w.mu.Lock()
		if id == "" {
			id = w.defaultID
		}
		e := w.entries[id]
		if w.closed {
			w.mu.Unlock()
			return nil, nil, fault.New(diag.Unavailable, diag.UIWorkspaceClosed)
		}
		if e == nil {
			w.mu.Unlock()
			return nil, nil, fault.New(diag.NotFound, diag.UIWorkspaceUnknown)
		}
		if e.ready != nil {
			ready := e.ready
			w.mu.Unlock()
			select {
			case <-ready:
				continue
			case <-ctx.Done():
				return nil, nil, ctx.Err()
			}
		}
		if e.runtime.Service != nil {
			e.refs++
			e.used = time.Now()
			service := e.runtime.Service
			w.mu.Unlock()
			return service, sync.OnceFunc(func() { w.mu.Lock(); e.refs--; e.used = time.Now(); w.mu.Unlock() }), nil
		}
		e.ready = make(chan struct{})
		w.workers.Add(1)
		w.mu.Unlock()
		runtime, err := w.options.Open(w.ctx, e.value)
		if err == nil && (runtime.Service == nil || runtime.Close == nil) {
			err = fault.New(diag.Unavailable, diag.UIWorkspaceClosed)
		}
		w.mu.Lock()
		if w.closed && err == nil {
			err = fault.New(diag.Unavailable, diag.UIWorkspaceClosed)
		}
		if err == nil {
			e.runtime = runtime
			e.value.Error = ""
		} else {
			e.value.Error = fault.PublicMessage(err, nil)
		}
		close(e.ready)
		e.ready = nil
		w.mu.Unlock()
		if err != nil && runtime.Close != nil {
			err = errors.Join(err, runtime.Close())
		}
		w.workers.Done()
		if err != nil {
			return nil, nil, err
		}
	}
}

func (w *Workspaces) ResolvePath(path string) (string, error) { return w.options.Check(path) }

func (s *Service) workspaceActivity() (int, int) {
	s.mu.Lock()
	defer s.mu.Unlock()
	n := 0
	for _, r := range s.active {
		n += len(r.approvals)
	}
	return len(s.active), n
}
func (s *Service) SessionActivity(id string) (string, int) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if r := s.sessionRequest(id); r != nil {
		return r.task.ID, len(r.approvals)
	}
	return "", 0
}
