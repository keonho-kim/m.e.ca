// Purpose: Verify runtime observations are durable, ordered and closed after caller cancellation.
// Pattern: Observer contract tests.
// Usage: go test -race ./internal/agent/monitor.
// Related: internal/agent/monitor/observer.go.
package monitor

import (
	"context"
	"fmt"
	"testing"
)

func TestMonitorPublishesOnlyDurableFacts(t *testing.T) {
	var stored []Record
	observer := New(nil, func(_ context.Context, r Record) error {
		if r.Kind == "fail" {
			return fmt.Errorf("storage failed")
		}
		stored = append(stored, r)
		return nil
	})
	if err := observer.Observe(context.Background(), Record{RunID: "t", Kind: "proposed", CallID: "c"}); err != nil {
		t.Fatal(err)
	}
	if err := observer.Observe(context.Background(), Record{Kind: "fail"}); err == nil {
		t.Fatal("storage error ignored")
	}
	observer.Close()
	if len(stored) != 1 || stored[0].Sequence != 1 {
		t.Fatalf("records=%+v", stored)
	}
	if err := observer.Observe(context.Background(), Record{Kind: "late"}); err == nil {
		t.Fatal("closed observer accepted work")
	}
}

func TestCallerCancellationStopsPersistence(t *testing.T) {
	entered := make(chan struct{})
	stopped := make(chan struct{})
	observer := New(nil, func(ctx context.Context, _ Record) error {
		close(entered)
		<-ctx.Done()
		close(stopped)
		return ctx.Err()
	})
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	caller := make(chan error, 1)
	go func() { caller <- observer.Observe(ctx, Record{Kind: "proposed"}) }()
	<-entered
	cancel()
	<-stopped
	observer.Close()
	if err := <-caller; err == nil {
		t.Fatal("canceled write reported success")
	}
}
