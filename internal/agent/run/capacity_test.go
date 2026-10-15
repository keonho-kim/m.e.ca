// Purpose: Verify context-capacity recovery compacts once and never replays an executed tool.
// Pattern: Runtime integration test with deterministic provider and memory boundaries.
// Usage: go test ./internal/agent/run -run TestCapacity.
// Related: internal/agent/run/memory.go, internal/agent/compact/controller.go.
package run

import (
	"context"
	"encoding/json"
	"github.com/keonho-kim/v-code/internal/agent/compact"
	modelapi "github.com/keonho-kim/v-code/internal/agent/model"
	"github.com/keonho-kim/v-code/internal/agent/prompt"
	"strings"
	"testing"
)

type capacityFailure struct{}

func (capacityFailure) Error() string      { return "context capacity exceeded" }
func (capacityFailure) ContextLimit() bool { return true }

type capacityGenerator struct {
	base          *generator
	failed        bool
	summaries     int
	activeRequest string
}

func (g *capacityGenerator) Generate(ctx context.Context, r modelapi.Request, emit func(modelapi.Event) error) error {
	if handled, err := completionFixture(r, emit); handled {
		return err
	}
	if strings.HasPrefix(r.Messages[0].Content, "# Conversation memory") {
		g.summaries++
		if strings.Contains(r.Messages[1].Content, `"ActiveRequest":"Read a"`) {
			g.activeRequest = "Read a"
		}
		var text strings.Builder
		for _, section := range []string{"Requests", "Run history", "Steps and changed files", "Verification", "Decisions and constraints", "Pending work and blockers", "Continuation"} {
			text.WriteString("## " + section + "\n- The requested file was read; report the observation.\n")
		}
		return emit(modelapi.Event{Delta: text.String(), Done: true})
	}
	if len(g.base.requests) == 1 && !g.failed {
		g.failed = true
		return capacityFailure{}
	}
	return g.base.Generate(ctx, r, emit)
}

type capacityStore struct {
	journal
	memory compact.Memory
}

func (*capacityStore) AddContextTokens(context.Context, string, compact.TokenCount) error { return nil }
func (s *capacityStore) Memory(context.Context, string) (compact.Memory, error)           { return s.memory, nil }
func (s *capacityStore) MemoryWatermark(context.Context, string, int64) (compact.Watermark, error) {
	return compact.Watermark{Timeline: int64(len(s.messages)), Usage: int64(len(s.messages))}, nil
}
func (*capacityStore) CompactionBoundary(_ context.Context, _ string, through int64, _ int) (int64, error) {
	return through, nil
}
func (s *capacityStore) MemoryRecords(_ context.Context, _ string, after, through int64) ([]compact.Record, int64, error) {
	var records []compact.Record
	for i := after; i < through; i++ {
		body, err := json.Marshal(s.messages[i])
		if err != nil {
			return nil, 0, err
		}
		records = append(records, compact.Record{Sequence: i + 1, Role: s.messages[i].Role, Text: string(body)})
	}
	return records, through, nil
}
func (s *capacityStore) InstallMemory(_ context.Context, _ string, _ int, m compact.Memory) error {
	s.memory = m
	return nil
}
func (s *capacityStore) ContextMessages(_ context.Context, _ string, after int64, _ int) ([]modelapi.Message, error) {
	return append([]modelapi.Message(nil), s.messages[after:]...), nil
}

func TestCapacityCompactionDoesNotReplayRead(t *testing.T) {
	base := &generator{replies: []string{nativeCall("read_file", map[string]any{"path": "a"}), "Observed the file"}}
	g := &capacityGenerator{base: base}
	e := &executor{}
	store := &capacityStore{journal: journal{messages: []modelapi.Message{{Role: "user", Content: "Old request: implement a different project"}}}}
	r := fixture(base, e, &store.journal)
	r.Generator = g
	r.Journal = store
	r.Usage = &Usage{}
	r.Memory = &compact.Controller{Repository: store, Session: r.Session, Model: r.Model, Prompt: prompt.Defaults().Compaction, Settings: compact.Defaults(), Redact: r.Redact}
	result, err := r.Run(context.Background(), "Read a", approve, emit)
	if g.activeRequest != "Read a" {
		t.Fatal("active request missing from compaction input")
	}
	last := base.requests[len(base.requests)-1].Messages
	if !strings.HasSuffix(last[len(last)-1].Content, "Read a") {
		t.Fatal("current request missing after compaction")
	}
	if err != nil || result.Status != Answered || e.executed != 1 || g.summaries != 1 || r.Usage.ModelCalls != 5 {
		t.Fatalf("result=%+v executions=%d summaries=%d calls=%d err=%v", result, e.executed, g.summaries, r.Usage.ModelCalls, err)
	}
}
