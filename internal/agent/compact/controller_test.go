// Purpose: Verify token-triggered compaction publishes summary-only memory and never replaces valid memory on failure.
// Pattern: Compaction behavior tests.
// Usage: go test ./internal/agent/compact.
// Related: internal/agent/compact/controller.go, internal/agent/compact/context.go.
package compact

import (
	"context"
	"encoding/json"
	"fmt"
	modelapi "github.com/keonho-kim/v-code/internal/agent/model"
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
func (r *repository) CompactionBoundary(_ context.Context, _ string, through int64, _ int) (int64, error) {
	return through, nil
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
func summary() string { return "Preserve task state and source references." }
func TestThresholdPublishesSummaryOnly(t *testing.T) {
	repo := &repository{mark: Watermark{Tokens: 199999, Usage: 10, Timeline: 2}, records: []Record{{Sequence: 1, Role: "user", Text: "goal"}, {Sequence: 2, Role: "outcome", Text: "edit_file tool_succeeded src/auth.go"}}}
	controller := Controller{Repository: repo, Session: "s", Model: "m", Prompt: prompt.Defaults().Compaction, Settings: Defaults(), Redact: func(s string) string { return s }}
	calls := 0
	generate := func(_ context.Context, r modelapi.Request, emit func(modelapi.Event) error) error {
		calls++
		if len(r.Messages) != 2 {
			t.Fatal("invalid compaction context")
		}
		if err := emit(modelapi.Event{Delta: summary()}); err != nil {
			return err
		}
		return emit(modelapi.Event{Done: true})
	}
	changed, err := controller.Refresh(context.Background(), generate)
	if err != nil || changed || calls != 0 {
		t.Fatalf("early trigger: %v %v", changed, err)
	}
	repo.mark.Tokens = 200000
	changed, err = controller.Refresh(context.Background(), generate)
	if err != nil || !changed || repo.installed != 1 {
		t.Fatalf("trigger: %v %v", changed, err)
	}
	body, marshalErr := json.Marshal(repo.memory)
	if marshalErr != nil || strings.Contains(string(body), "Ledger") || strings.Contains(string(body), "src/auth.go") {
		t.Fatalf("execution ledger persisted in memory: %s %v", body, marshalErr)
	}
	memory, through, err := controller.Context()
	if through != repo.memory.Through {
		t.Fatal("compaction boundary lost")
	}
	if err != nil || memory != summary() {
		t.Fatalf("unexpected compacted context: %s %v", memory, err)
	}
}
func TestEmptySummaryPreservesPreviousMemory(t *testing.T) {
	repo := &repository{memory: Memory{Revision: 1, Markdown: summary()}, mark: Watermark{Tokens: 200000, Usage: 2, Timeline: 1}, records: []Record{{Sequence: 1, Role: "user", Text: "goal"}}}
	controller := Controller{Repository: repo, Session: "s", Model: "m", Prompt: prompt.Defaults().Compaction, Settings: Defaults(), Redact: func(s string) string { return s }}
	_, err := controller.Refresh(context.Background(), func(_ context.Context, _ modelapi.Request, emit func(modelapi.Event) error) error {
		if err := emit(modelapi.Event{Delta: "   \n"}); err != nil {
			return err
		}
		return emit(modelapi.Event{Done: true})
	})
	if err == nil || repo.installed != 0 || repo.memory.Revision != 1 {
		t.Fatal("empty summary replaced the evidence memory")
	}
}

func TestCancellationDoesNotAdvanceMemoryWatermark(t *testing.T) {
	repo := &repository{mark: Watermark{Tokens: 200000, Usage: 2, Timeline: 1}, records: []Record{{Sequence: 1, Role: "user", Text: "goal"}}}
	controller := Controller{Repository: repo, Session: "s", Model: "m", Prompt: prompt.Defaults().Compaction, Settings: Defaults(), Redact: func(s string) string { return s }}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	_, err := controller.Refresh(ctx, func(ctx context.Context, _ modelapi.Request, _ func(modelapi.Event) error) error {
		cancel()
		return ctx.Err()
	})
	if err == nil || repo.installed != 0 || repo.memory.UsageThrough != 0 || repo.memory.Through != 0 {
		t.Fatal("cancelled compaction advanced source or token watermark")
	}
}

func TestThresholdCollectsSourceRecordsOnce(t *testing.T) {
	repo := &repository{mark: Watermark{Tokens: 200000, Usage: 2, Timeline: 2}, records: []Record{{Sequence: 1, Role: "user", Text: "first instruction"}, {Sequence: 2, Role: "assistant", Text: "second observation"}}}
	c := Controller{Repository: repo, Session: "s", Model: "m", Prompt: prompt.Defaults().Compaction, Settings: Defaults(), Redact: func(s string) string { return s }}
	calls := 0
	changed, err := c.Refresh(context.Background(), func(_ context.Context, r modelapi.Request, emit func(modelapi.Event) error) error {
		calls++
		if !strings.Contains(r.Messages[1].Content, "first instruction") || !strings.Contains(r.Messages[1].Content, "second observation") {
			t.Fatal("source missing from batch")
		}
		return emit(modelapi.Event{Delta: summary(), Done: true})
	})
	if err != nil || !changed || calls != 1 {
		t.Fatalf("changed=%v calls=%d err=%v", changed, calls, err)
	}
}

func TestSummaryFormattingIsGuidanceOnly(t *testing.T) {
	for _, text := range []string{
		"The requested change is implemented. Tests are pending.",
		"## Continuation\nRun tests.\n## Requests\nFix the parser.",
		"- Changed a.go\n- Pending: verify behavior",
		"### 작업 내역\n1. a.go 수정\n2. 검증 필요",
		"```markdown\nSummary in a fence.\n```",
	} {
		t.Run(text, func(t *testing.T) {
			repo := &repository{memory: Memory{Revision: 1, Markdown: summary()}, mark: Watermark{Tokens: 200000, Usage: 2, Timeline: 1}, records: []Record{{Sequence: 1, Role: "user", Text: "fix parser"}}}
			c := Controller{Repository: repo, Session: "s", Model: "m", Prompt: prompt.Defaults().Compaction, Settings: Defaults(), Redact: func(s string) string { return s }}
			changed, err := c.Refresh(context.Background(), func(_ context.Context, _ modelapi.Request, emit func(modelapi.Event) error) error {
				return emit(modelapi.Event{Delta: text, Done: true})
			})
			if err != nil || !changed || repo.installed != 1 || repo.memory.Markdown != text {
				t.Fatalf("format rejected or rewritten: changed=%t err=%v", changed, err)
			}
		})
	}
}
