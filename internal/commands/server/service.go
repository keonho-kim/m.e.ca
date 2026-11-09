// Purpose: Own server state and process-scoped dependencies without performing I/O under its mutex.
// Pattern: Application service with explicit resource ownership.
// Usage: Transport commands call use cases; lifecycle.go closes all owned operations and sessions.
// Related: internal/commands/server/operations.go, internal/commands/server/lifecycle.go.
package server

import (
	"context"
	"sync"

	"github.com/keonho-kim/meca/internal/tools/workspacegate"

	diagcatalog "github.com/keonho-kim/meca/internal/diagnostics/catalog"
	faults "github.com/keonho-kim/meca/internal/diagnostics/fault"
)

type Service struct {
	workspace     workspacegate.Gate
	mu            sync.Mutex
	ctx           context.Context
	cancel        context.CancelFunc
	options       Options
	conversation  *Conversation
	jobs          map[string]*Conversation
	attachments   map[string]*attachment
	active        map[string]*request
	bySession     map[string]*request
	admissions    chan struct{}
	last          *Task
	changing      bool
	admitting     string
	closed        bool
	closeDone     chan struct{}
	closeErr      error
	completionErr error
	executionErr  error
	workers       sync.WaitGroup
	feed          eventLog
}

func New(ctx context.Context, options Options) (*Service, error) {
	if options.Concurrency == 0 {
		options.Concurrency = 1
	}
	if options.Concurrency < 1 {
		return nil, faults.New(diagcatalog.InvalidArgument, diagcatalog.ServerConcurrencyRange)
	}
	if options.Open == nil || options.List == nil || options.PrepareModel == nil || options.UpdateConfig == nil || options.Ledger == nil {
		return nil, faults.New(diagcatalog.InvalidArgument, diagcatalog.ServerServerIODependenciesAreRequired)
	}
	ctx, cancel := context.WithCancel(ctx)
	return &Service{ctx: ctx, cancel: cancel, options: options, closeDone: make(chan struct{}), feed: newEventLog(), active: make(map[string]*request), jobs: make(map[string]*Conversation), admissions: make(chan struct{}, 1)}, nil
}
