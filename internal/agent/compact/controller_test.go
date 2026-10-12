// Purpose: Verify token-triggered compaction preserves runtime facts and never replaces valid memory on failure.
// Pattern: Compaction behavior tests.
// Usage: go test ./internal/agent/compact.
// Related: internal/agent/compact/controller.go, internal/agent/compact/ledger.go.
package compact

import (
	"context"
	"fmt"
	"github.com/keonho-kim/v-code/internal/agent"
	"github.com/keonho-kim/v-code/internal/agent/prompt"
	"strings"
	"testing"
)

type repository struct {
	memory    Memory
	mark      Watermark
	records   []Record
	installed int
}

func (*repository) AddContextTokens(context.Context, string, TokenCount) error { return nil }
func (r *repository) Memory(context.Context, string) (Memory, error)           { return r.memory, nil }
func (r *repository) MemoryWatermark(context.Context, string, int64) (Watermark, error) {
	return r.mark, nil
}
func (r *repository) MemoryRecords(_ context.Context, _ string, after, through int64) ([]Record, int64, error) {
	var out []Record
	for _, record := range r.records {
		if record.Sequence > after && record.Sequence <= through {
			out = append(out, record)
		}
	}
	return out, through, nil
}
func (r *repository) InstallMemory(_ context.Context, _ string, expected int, next Memory) error {
	if expected != r.memory.Revision {
		return fmt.Errorf("stale")
	}
	r.memory = next
	r.installed++
	return nil
}
func summary() string {
	var b strings.Builder
	for _, section := range sections {
		b.WriteString("## " + section + "\n- Preserve task state and source references.\n")
	}
	return b.String()
}
func TestThresholdAndFileLedger(t *testing.T) {
	repo := &repository{mark: Watermark{Tokens: 255999, Usage: 10, Timeline: 2}, records: []Record{{Sequence: 1, Role: "task", Text: "goal", Task: &TaskFact{ID: "task", Objective: "goal", Status: "running", Step: &StepRef{PlanID: "plan", Revision: 2, Index: 0, Description: "edit"}}}, {Sequence: 2, Role: "operation", Operation: &Operation{TaskID: "task", CallID: "call", Name: "edit_file", Status: "tool_succeeded", Changes: []agent.FileChange{{Action: "edit_file", Path: "src/auth.go"}}}}}}
	controller := Controller{Repository: repo, Session: "s", Model: "m", Prompt: prompt.Defaults().Compaction, Settings: Defaults(), Redact: func(s string) string { return s }}
	calls := 0
	generate := func(_ context.Context, r agent.ModelRequest, emit func(agent.ModelEvent) error) error {
		calls++
		if len(r.Messages) != 2 {
			t.Fatal("invalid compaction context")
		}
		if err := emit(agent.ModelEvent{Delta: summary()}); err != nil {
			return err
		}
		return emit(agent.ModelEvent{Done: true})
	}
	changed, err := controller.Refresh(context.Background(), generate)
	if err != nil || changed || calls != 0 {
		t.Fatalf("early trigger: %v %v", changed, err)
	}
	repo.mark.Tokens = 256000
	changed, err = controller.Refresh(context.Background(), generate)
	if err != nil || !changed || repo.installed != 1 {
		t.Fatalf("trigger: %v %v", changed, err)
	}
	memory, err := controller.Context()
	if err != nil || !strings.Contains(memory, "src/auth.go") || !strings.Contains(memory, "revision 2 step 1") {
		t.Fatalf("lost facts: %s %v", memory, err)
	}
}
func TestInvalidSummaryPreservesPreviousMemory(t *testing.T) {
	repo := &repository{memory: Memory{Revision: 1, Markdown: summary()}, mark: Watermark{Tokens: 256000, Usage: 2, Timeline: 1}, records: []Record{{Sequence: 1, Role: "user", Text: "goal"}}}
	controller := Controller{Repository: repo, Session: "s", Model: "m", Prompt: prompt.Defaults().Compaction, Settings: Defaults(), Redact: func(s string) string { return s }}
	_, err := controller.Refresh(context.Background(), func(_ context.Context, _ agent.ModelRequest, emit func(agent.ModelEvent) error) error {
		if err := emit(agent.ModelEvent{Delta: "I think everything is done."}); err != nil {
			return err
		}
		return emit(agent.ModelEvent{Done: true})
	})
	if err == nil || repo.installed != 0 || repo.memory.Revision != 1 {
		t.Fatal("invalid prose replaced the evidence memory")
	}
}

func TestCancellationDoesNotAdvanceMemoryWatermark(t *testing.T) {
	repo := &repository{mark: Watermark{Tokens: 256000, Usage: 2, Timeline: 1}, records: []Record{{Sequence: 1, Role: "user", Text: "goal"}}}
	controller := Controller{Repository: repo, Session: "s", Model: "m", Prompt: prompt.Defaults().Compaction, Settings: Defaults(), Redact: func(s string) string { return s }}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	_, err := controller.Refresh(ctx, func(ctx context.Context, _ agent.ModelRequest, _ func(agent.ModelEvent) error) error {
		cancel()
		return ctx.Err()
	})
	if err == nil || repo.installed != 0 || repo.memory.UsageThrough != 0 || repo.memory.Through != 0 {
		t.Fatal("cancelled compaction advanced source or token watermark")
	}
}
