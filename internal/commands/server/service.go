// Purpose: Own server state and process-scoped dependencies without performing I/O under its mutex.
// Pattern: Application service with explicit resource ownership.
// Usage: Transport commands call use cases; lifecycle.go closes all owned operations and sessions.
// Related: internal/commands/server/operations.go, internal/commands/server/lifecycle.go.
package server

import (
	"context"
	"sync"
)

type Service struct {
	mu           sync.Mutex
	ctx          context.Context
	cancel       context.CancelFunc
	options      Options
	scope        string
	conversation *Conversation
	job          *Conversation
	active       *request
	last         *Task
	changing     bool
	admitting    string
	submitting   string
	closed       bool
	closeDone    chan struct{}
	closeErr     error
	workers      sync.WaitGroup
	feed         eventLog
}

func New(ctx context.Context, options Options) (*Service, error) {
	if options.Open == nil || options.List == nil || options.PrepareModel == nil || options.UpdateConfig == nil || options.Ledger == nil {
		return nil, failure("invalid_argument", "server I/O dependencies are required")
	}
	ctx, cancel := context.WithCancel(ctx)
	return &Service{ctx: ctx, cancel: cancel, options: options, scope: ProjectScope(options.Project), closeDone: make(chan struct{}), feed: newEventLog()}, nil
}
