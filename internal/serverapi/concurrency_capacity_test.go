// Purpose: Honor operator-selected task concurrency above eight without preallocating by the configured maximum.
// Pattern: Channel-controlled native execution and wire contract tests.
// Usage: go test -race ./internal/serverapi -run 'TestConcurrentTasksAboveEight|TestConcurrencyState'.
// Related: internal/commands/server/service.go, internal/commands/server/task_registry.go, api/meca/v1/state.proto.
package serverapi

import (
	"context"
	"fmt"
	"testing"

	"github.com/keonho-kim/meca/internal/agent/model"
	"github.com/keonho-kim/meca/internal/commands/server"
	diagcatalog "github.com/keonho-kim/meca/internal/diagnostics/catalog"
	faults "github.com/keonho-kim/meca/internal/diagnostics/fault"
	v1 "github.com/keonho-kim/meca/internal/serverapi/gen/mecav1"
	"google.golang.org/protobuf/encoding/protojson"
)

func TestConcurrentTasksAboveEightHonorConfiguredCapacity(t *testing.T) {
	const limit = 9
	started := make(chan struct{}, limit)
	f := newFixture(t, generateFunc(func(ctx context.Context, _ model.Request, _ func(model.Event) error) error {
		started <- struct{}{}
		<-ctx.Done()
		return ctx.Err()
	}), false, limit)
	var tasks []server.Task
	for i := range limit {
		task, _ := startNativeTask(t, f, fmt.Sprintf("parallel-%d", i), "wait")
		tasks = append(tasks, task)
	}
	ctx := deadline(t)
	for range limit {
		select {
		case <-started:
		case <-ctx.Done():
			t.Fatal(ctx.Err())
		}
	}
	state := f.app.State()
	if state.Concurrency != limit || len(state.Tasks) != limit {
		t.Fatal("configured tasks were not running", state.Concurrency, len(state.Tasks))
	}
	if _, err := f.app.Start(ctx, state.Session.ID, "capacity-overflow", "wait"); faults.Code(err) != diagcatalog.ResourceExhausted {
		t.Fatal("operator limit was ignored", err)
	}
	for _, task := range tasks {
		if err := f.app.Cancel(task.ID); err != nil {
			t.Fatal(err)
		}
	}
}

func TestConcurrencyStatePreservesNativeIntegerRange(t *testing.T) {
	limit := int(^uint(0) >> 1)
	f := newFixture(t, generateFunc(idleGenerator), false, limit)
	state := snapshot(f.app.State())
	if int64(state.GetConcurrency()) != int64(limit) {
		t.Fatal("state truncated configured concurrency", state.GetConcurrency(), limit)
	}
	body, err := protojson.Marshal(state)
	if err != nil {
		t.Fatal(err)
	}
	var restored v1.Snapshot
	if err := protojson.Unmarshal(body, &restored); err != nil || restored.GetConcurrency() != int64(limit) {
		t.Fatal("wire response changed concurrency", restored.GetConcurrency(), err)
	}
}
