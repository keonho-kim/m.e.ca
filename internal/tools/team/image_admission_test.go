// Purpose: Verify image memory admission remains shared, cancellable and independent of write leases.
// Pattern: Deterministic concurrent operation tests.
// Usage: go test -race ./internal/tools/team -run TestImageAdmission.
// Related: internal/tools/team/local.go, internal/commands/teamrequest/request.go.
package team

import (
	"context"
	"errors"
	"sync"
	"testing"

	toolapi "github.com/keonho-kim/meca/internal/agent/tool"
	"github.com/keonho-kim/meca/internal/tools/team/approval"
)

type imageExecution struct{ entered, release chan struct{} }

func (op imageExecution) Approval() toolapi.Approval { return toolapi.Approval{} }
func (op imageExecution) Execute(context.Context, bool) (toolapi.Result, error) {
	close(op.entered)
	<-op.release
	return toolapi.Result{}, nil
}

func TestImageAdmissionCancellationDoesNotStartAnotherDecoder(t *testing.T) {
	executor := &Executor{Images: make(chan struct{}, 1), Refusals: &approval.Ledger{}}
	first := imageExecution{entered: make(chan struct{}), release: make(chan struct{})}
	release := sync.OnceFunc(func() { close(first.release) })
	var workers sync.WaitGroup
	workers.Add(1)
	defer func() { release(); workers.Wait() }()
	done := make(chan error, 1)
	go func() {
		defer workers.Done()
		_, err := (&localOperation{executor: executor, call: toolapi.Call{Name: "view_image"}, operation: first}).Execute(context.Background(), true)
		done <- err
	}()
	<-first.entered
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	second := imageExecution{entered: make(chan struct{}), release: make(chan struct{})}
	_, err := (&localOperation{executor: executor, call: toolapi.Call{Name: "view_image"}, operation: second}).Execute(ctx, true)
	if !errors.Is(err, context.Canceled) {
		t.Fatal(err)
	}
	select {
	case <-second.entered:
		t.Fatal("second decoder exceeded the shared image budget")
	default:
	}
	release()
	if err := <-done; err != nil || len(executor.Images) != 0 {
		t.Fatal("image completion retained a permit", err)
	}
}
