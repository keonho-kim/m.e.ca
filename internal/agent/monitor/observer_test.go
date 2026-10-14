// Purpose: Verify background observations are durable, ordered and joined on shutdown.
// Pattern: Observer contract tests.
// Usage: go test -race ./internal/agent/monitor.
// Related: internal/agent/monitor/observer.go.
package monitor

import (
	"context"
	"fmt"
	"strings"
	"testing"
)

func TestMonitorPublishesOnlyDurableFacts(t *testing.T) {
	var stored []Record
	observer := New(context.Background(), nil, func(_ context.Context, r Record) error {
		if r.Kind == "fail" {
			return fmt.Errorf("storage failed")
		}
		stored = append(stored, r)
		return nil
	})
	if err := observer.Observe(context.Background(), Record{GoalID: "g", TaskID: "t", Kind: "proposed", CallID: "c"}); err != nil {
		t.Fatal(err)
	}
	if err := observer.Observe(context.Background(), Record{Kind: "fail"}); err == nil {
		t.Fatal("storage error ignored")
	}
	snapshot, err := observer.Snapshot(context.Background())
	if err != nil || !strings.Contains(snapshot, "proposed") || strings.Contains(snapshot, `"Kind":"fail"`) {
		t.Fatalf("snapshot=%s %v", snapshot, err)
	}
	observer.Close()
	if len(stored) != 1 || stored[0].Sequence != 1 {
		t.Fatalf("records=%+v", stored)
	}
	if err := observer.Observe(context.Background(), Record{Kind: "late"}); err == nil {
		t.Fatal("closed observer accepted work")
	}
}

func TestCloseCancelsActivePersistence(t *testing.T) {
	entered := make(chan struct{})
	stopped := make(chan struct{})
	observer := New(context.Background(), nil, func(ctx context.Context, _ Record) error {
		close(entered)
		<-ctx.Done()
		close(stopped)
		return ctx.Err()
	})
	caller := make(chan error, 1)
	go func() { caller <- observer.Observe(context.Background(), Record{Kind: "proposed"}) }()
	<-entered
	observer.Close()
	<-stopped
	if err := <-caller; err == nil {
		t.Fatal("canceled write reported success")
	}
}
