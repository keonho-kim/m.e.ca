// Purpose: Verify pending-command completion, in-flight observations and cancellation durability.
// Pattern: Runtime resource lifecycle regression tests.
// Usage: go test -race ./internal/agent/run -run TestCommandLifecycle.
// Related: internal/agent/run/commands.go, internal/agent/run/command_generation.go.
package run

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"testing/synctest"
	"time"

	modelapi "github.com/keonho-kim/meca/internal/agent/model"
	runrecord "github.com/keonho-kim/meca/internal/agent/record"
	toolapi "github.com/keonho-kim/meca/internal/agent/tool"
)

type commandFixture struct {
	*executor
	mu        sync.Mutex
	active    bool
	completed bool
	stopped   bool
}

func (e *commandFixture) BeginCommands(context.Context, time.Time) {
	e.mu.Lock()
	e.active = true
	e.mu.Unlock()
}
func (e *commandFixture) StopCommands() {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.stopped = true
	if e.active {
		e.active = false
		e.completed = true
	}
}
func (e *commandFixture) CommandActivity() []toolapi.CommandRun {
	e.mu.Lock()
	defer e.mu.Unlock()
	if e.active {
		return []toolapi.CommandRun{{ID: "cmd_owned", Command: "test", Status: "running"}}
	}
	return nil
}
func (e *commandFixture) CollectCommands() ([]toolapi.CommandCompletion, error) {
	e.mu.Lock()
	defer e.mu.Unlock()
	if !e.completed {
		return nil, nil
	}
	return []toolapi.CommandCompletion{{ID: "cmd_owned", CallID: "call_original", Command: "test", Result: toolapi.Result{
		Started: true, Summary: "observed final output", Content: json.RawMessage("{\"Status\":\"exited\",\"ExitCode\":0,\"Stdout\":\"actual result\"}"),
		Process: &toolapi.ProcessOutcome{CommandID: "cmd_owned", Command: "test", Status: "exited", ExitCode: 0},
	}}}, nil
}
func (e *commandFixture) AcknowledgeCommand(string) error {
	e.mu.Lock()
	e.completed = false
	e.mu.Unlock()
	return nil
}

func (e *commandFixture) CommandCapture(string) (json.RawMessage, error) { return nil, nil }

func TestCommandLifecycleFeedbackExplainsManagementTools(t *testing.T) {
	j := &journal{}
	e := &commandFixture{executor: &executor{}, active: true}
	r := fixture(&generator{}, e.executor, j)
	r.Tools = e
	s := execution{runner: r}
	pending, err := s.pendingCommands(context.Background())
	if err != nil || !pending || len(j.messages) != 1 {
		t.Fatalf("pending command guidance not recorded: pending=%v messages=%d error=%v", pending, len(j.messages), err)
	}
	for _, value := range []string{"cmd_owned", "wait_command", "cancel_command", "list_command_runs", "command_id"} {
		if !strings.Contains(j.messages[0].Content, value) {
			t.Fatalf("model feedback omitted %s: %s", value, j.messages[0].Content)
		}
	}
	if !e.active || e.executed != 0 {
		t.Fatal("guidance changed or replayed the running command")
	}
}

func TestCommandLifecycleRechecksResultsBeforeFinalAnswer(t *testing.T) {
	j := &journal{}
	e := &commandFixture{executor: &executor{}}
	r := fixture(&generator{}, e.executor, j)
	r.Tools = e
	calls := 0
	r.Generator = statisticsModel(func(_ context.Context, request modelapi.Request, send func(modelapi.Event) error) error {
		calls++
		if calls == 1 {
			e.mu.Lock()
			e.active = false
			e.completed = true
			e.mu.Unlock()
			return send(modelapi.Event{Delta: "premature response", Done: true})
		}
		found := false
		for _, message := range request.Messages {
			found = found || strings.Contains(message.Content, "actual result")
		}
		if !found {
			t.Error("completion absent from the next model context")
		}
		return send(modelapi.Event{Delta: "verified final response", Done: true})
	})
	result, err := r.Run(context.Background(), "run the command", approve, emit)
	if err != nil || result.Text != "verified final response" || calls != 2 || !e.stopped {
		t.Fatalf("result %+v calls=%d: %v", result, calls, err)
	}
	count := 0
	for _, event := range j.events {
		if event.Kind == "command_finished" {
			count++
		}
	}
	if count != 1 {
		t.Fatalf("completion recorded %d times", count)
	}
}

func TestCommandLifecycleCancellationStillCommitsCompletion(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	j := &journal{}
	e := &commandFixture{executor: &executor{}}
	r := fixture(&generator{}, e.executor, j)
	r.Tools = e
	r.Generator = statisticsModel(func(ctx context.Context, _ modelapi.Request, _ func(modelapi.Event) error) error {
		cancel()
		return ctx.Err()
	})
	_, err := r.Run(ctx, "run command", approve, func(runrecord.Event) error {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		return nil
	})
	if !errors.Is(err, context.Canceled) || errors.Is(err, errRuntimeRecord) {
		t.Fatalf("cancellation misreported: %v", err)
	}
	found := false
	for _, event := range j.events {
		found = found || event.Kind == "command_finished"
	}
	if !found || !e.stopped {
		t.Fatal("owned process completion lost on cancellation")
	}
}

func TestCommandLifecycleIndependentCommandsStartTogether(t *testing.T) {
	var calls []toolapi.Call
	for index, command := range []string{
		"printf ready > a.started; while [ ! -f b.started ]; do sleep 0.01; done",
		"printf ready > b.started; while [ ! -f a.started ]; do sleep 0.01; done",
	} {
		body, _ := json.Marshal(map[string]string{"command": command})
		calls = append(calls, toolapi.Call{ID: []string{"call_a", "call_b"}[index], Name: "execute_command", Arguments: body})
	}
	r, _, _, j, _ := realBatchRunner(t, calls)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	result, err := r.Run(ctx, "run both independent commands", approve, emit)
	if err != nil || result.Status != runrecord.Answered {
		t.Fatalf("commands did not overlap: %+v %v", result, err)
	}
	completed := 0
	for _, event := range j.events {
		if event.Kind == "tool_succeeded" && event.Operation != nil && event.Operation.Process != nil && event.Operation.Process.Status == "exited" {
			completed++
		}
	}
	if completed != 2 {
		t.Fatalf("recorded %d terminal results", completed)
	}
}

func TestCommandLifecycleCollectsDuringQuietGeneration(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		j := &journal{}
		e := &commandFixture{executor: &executor{}}
		r := fixture(&generator{}, e.executor, j)
		r.Tools = e
		var observed atomic.Bool
		calls := 0
		r.Generator = statisticsModel(func(_ context.Context, _ modelapi.Request, send func(modelapi.Event) error) error {
			calls++
			if calls == 1 {
				e.mu.Lock()
				e.active = false
				e.completed = true
				e.mu.Unlock()
				time.Sleep(3 * time.Second)
				if !observed.Load() {
					t.Error("completion waited for model output instead of being committed while idle")
				}
			}
			return send(modelapi.Event{Delta: "final response", Done: true})
		})
		result, err := r.Run(context.Background(), "wait for the command", approve, func(event runrecord.Event) error {
			if event.Kind == "command_finished" {
				observed.Store(true)
			}
			return nil
		})
		if err != nil || result.Status != runrecord.Answered || calls != 2 {
			t.Fatalf("result %+v calls=%d: %v", result, calls, err)
		}
	})
}
