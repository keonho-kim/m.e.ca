// Purpose: Verify shared token aggregates count concurrent append-only tails once and detach returned charges.
// Pattern: Repository lifecycle and concurrency regression tests.
// Usage: go test -race ./internal/storage/sqlite -run TestUsageAcross.
// Related: internal/storage/sqlite/usage.go, internal/storage/sqlite/usage_cache.go.
package sqlite

import (
	"context"
	"path/filepath"
	"sync"
	"testing"

	"github.com/keonho-kim/meca/internal/agent/model"
	"github.com/keonho-kim/meca/internal/agent/record"
)

func TestUsageAcrossReadHandlesAndConcurrentReaders(t *testing.T) {
	ctx := t.Context()
	project := t.TempDir()
	sessions, err := OpenSessions(filepath.Join(t.TempDir(), "sessions"))
	if err != nil {
		t.Fatal(err)
	}
	writer, err := sessions.Create(ctx, project, project)
	if err != nil {
		t.Fatal(err)
	}
	defer writer.Close()
	id := writer.Session().ID
	appendUsage := func() {
		t.Helper()
		usage := model.TokenUsage{Input: 13, Output: 7, Cost: new(0.5)}
		if err := writer.RecordEvent(ctx, id, record.Event{Kind: "usage", Usage: &usage}); err != nil {
			t.Fatal(err)
		}
	}
	appendUsage()
	total, err := sessions.TokenUsage(ctx, project, id)
	if err != nil || total.Cost == nil {
		t.Fatalf("first usage: %+v %v", total, err)
	}
	*total.Cost = 999
	appendUsage()
	var readers sync.WaitGroup
	for range 8 {
		readers.Go(func() {
			total, err := sessions.TokenUsage(ctx, project, id)
			if err != nil || total.Input != 26 || total.Output != 14 || total.Cost == nil || *total.Cost != 1 {
				t.Errorf("concurrent restored usage: %+v %v", total, err)
			}
		})
	}
	readers.Wait()
	canceled, cancel := context.WithCancel(ctx)
	cancel()
	if _, err := writer.TokenUsage(canceled, id); err == nil {
		t.Fatal("canceled usage read succeeded")
	}
	appendUsage()
	total, err = writer.TokenUsage(ctx, id)
	if err != nil || total.Input != 39 || total.Output != 21 || *total.Cost != 1.5 {
		t.Fatalf("canceled read advanced totals: %+v %v", total, err)
	}
}
