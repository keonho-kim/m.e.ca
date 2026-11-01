// Purpose: Preserve one durable thinking trace while retaining native reasoning locally.
// Pattern: Audit ownership regression tests.
// Usage: Run TestThinkingTrace in the transcript package.
// Related: internal/agent/lat/transcript/journal.go, internal/agent/run/generate.go.
package transcript

import (
	"context"
	"errors"
	"testing"

	modelapi "github.com/keonho-kim/meca/internal/agent/model"
	runrecord "github.com/keonho-kim/meca/internal/agent/record"
)

type auditRecorder struct {
	events  []runrecord.Event
	failure error
}

func (r *auditRecorder) Commit(_ context.Context, _ string, entries []runrecord.Entry) error {
	if r.failure != nil {
		return r.failure
	}
	for _, entry := range entries {
		r.events = append(r.events, *entry.Event)
	}
	return nil
}

func TestThinkingTraceHasOneDurableOwner(t *testing.T) {
	ctx := context.Background()
	parent := &auditRecorder{}
	j, err := New(parent, "session", "parent_run", "plan")
	if err != nil {
		t.Fatal(err)
	}
	defer j.Close()
	if err := j.RecordEvent(ctx, "session", runrecord.Event{Kind: "thinking", Thinking: "native thought"}); err != nil {
		t.Fatal(err)
	}
	message := modelapi.Message{Role: "assistant", Content: "planned work", Reasoning: "native thought", ReasoningState: []byte("provider continuation")}
	if err := j.Commit(ctx, "session", []runrecord.Entry{{Message: &message}}); err != nil {
		t.Fatal(err)
	}
	if len(parent.events) != 2 || parent.events[0].Thinking != "native thought" || parent.events[1].Message.Reasoning != "" {
		t.Fatal("durable trace lost or duplicated", parent.events)
	}
	native, err := j.ContextMessages(ctx, "session", 0, 4096, true)
	if err != nil || native[0].Reasoning != message.Reasoning || string(native[0].ReasoningState) != string(message.ReasoningState) {
		t.Fatalf("native reasoning or continuation changed: %+v %v", native, err)
	}
	if j.thinking != "" {
		t.Fatal("completed thinking remained separately retained")
	}
	if err := j.RecordEvent(ctx, "session", runrecord.Event{Kind: "thinking", Thinking: "interrupted thought"}); err != nil {
		t.Fatal(err)
	}
	j.Close()
	if parent.events[2].Thinking != "interrupted thought" {
		t.Fatal("interrupted thinking depended on a later assistant message")
	}
}

func TestThinkingTraceFailureDoesNotClaimDurability(t *testing.T) {
	parent := &auditRecorder{failure: errors.New("disk unavailable")}
	j, err := New(parent, "session", "parent_run", "review")
	if err != nil {
		t.Fatal(err)
	}
	defer j.Close()
	if err := j.RecordEvent(context.Background(), "session", runrecord.Event{Kind: "thinking", Thinking: "not durable"}); !errors.Is(err, parent.failure) || j.thinking != "" {
		t.Fatal("failed thinking event changed ownership", err)
	}
	parent.failure = nil
	message := modelapi.Message{Role: "assistant", Reasoning: "not durable", Content: "report"}
	if err := j.Commit(context.Background(), "session", []runrecord.Entry{{Message: &message}}); err != nil {
		t.Fatal(err)
	}
	if parent.events[0].Message.Reasoning != message.Reasoning {
		t.Fatal("unrecorded thinking was silently discarded")
	}
}
