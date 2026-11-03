// Purpose: Verify cooperative input reaches decisions after compaction and defers stale user-directed effects.
// Pattern: Scripted runtime boundary tests.
// Usage: go test ./internal/agent/run -run TestCooperativeContext.
// Related: internal/agent/run/context_inbox.go, internal/agent/run/loop.go.
package run

import (
	"context"
	"strings"
	"testing"

	"github.com/keonho-kim/meca/internal/agent/compact"
	modelapi "github.com/keonho-kim/meca/internal/agent/model"
	"github.com/keonho-kim/meca/internal/agent/prompt"
	runrecord "github.com/keonho-kim/meca/internal/agent/record"
)

type contextJournalFixture struct {
	journal
	cursor int64
}

func (j *contextJournalFixture) CommitContext(ctx context.Context, id string, entries []runrecord.Entry, cursor int64) error {
	if len(entries) > 0 {
		if err := j.Commit(ctx, id, entries); err != nil {
			return err
		}
	}
	j.cursor = cursor
	return nil
}

func TestCooperativeContextDefersToolForNewUserInstruction(t *testing.T) {
	g := &generator{replies: []string{nativeCall("write_file", map[string]any{"path": "obsolete.txt", "content": "old"}), "Understood the new instruction"}}
	j := &contextJournalFixture{}
	tools := &executor{}
	r := fixture(g, tools, &j.journal)
	r.Journal = j
	r.PollContext = func(context.Context) (ContextBatch, error) {
		if len(g.requests) == 1 && j.cursor == 0 {
			return ContextBatch{Cursor: 1, Objective: "Do not create the file", Messages: []modelapi.Message{{Role: "user", Content: "Do not create the file"}}}, nil
		}
		return ContextBatch{}, nil
	}
	result, err := r.Run(context.Background(), "Create a file", approve, emit)
	if err != nil || result.Status != runrecord.Answered || tools.executed != 0 || tools.prepared != 0 {
		t.Fatalf("result=%+v executions=%d error=%v", result, tools.executed, err)
	}
	paired := false
	for _, message := range j.messages {
		paired = paired || message.Role == "tool" && strings.Contains(message.Content, "Nothing in this response executed")
	}
	if !paired {
		t.Fatal("deferred native call lost its paired result")
	}
}

type contextActivity struct {
	activityRepository
	cursor int64
}

func (j *contextActivity) CommitContext(ctx context.Context, id string, entries []runrecord.Entry, cursor int64) error {
	if len(entries) > 0 {
		if err := j.Commit(ctx, id, entries); err != nil {
			return err
		}
	}
	j.cursor = cursor
	return nil
}
func TestCooperativeContextArrivingDuringCompactionReachesFirstCall(t *testing.T) {
	j := &contextActivity{}
	j.messages = []modelapi.Message{{Role: "user", Content: "Earlier requirement"}}
	r := fixture(&generator{}, &executor{}, &j.journal)
	r.Journal = j
	r.Memory = &compact.Controller{Repository: j, Session: r.Session, Model: r.Model, Prompt: prompt.Defaults().Compaction, Settings: compact.Defaults(), Redact: r.Redact}
	pending := false
	r.PollContext = func(context.Context) (ContextBatch, error) {
		if pending && j.cursor == 0 {
			return ContextBatch{Cursor: 9, Messages: []modelapi.Message{{ID: "peer:agent-b/9", Role: "system", Content: "Public conversation — agent-b\nUse the existing storage interface."}}}, nil
		}
		return ContextBatch{}, nil
	}
	r.Generator = boundaryModel(func(_ context.Context, request modelapi.Request, emit func(modelapi.Event) error) error {
		if len(request.Tools) == 0 {
			pending = true
			return emit(modelapi.Event{Delta: "- Previous changes retained.", Done: true})
		}
		found := false
		for _, message := range request.Messages {
			found = found || strings.Contains(message.Content, "Use the existing storage interface")
		}
		if !found {
			t.Error("first post-compaction call missed the new conversation")
		}
		return emit(modelapi.Event{Delta: "done", Done: true})
	})
	if _, err := r.Run(context.Background(), "Continue implementation", approve, emit); err != nil {
		t.Fatal(err)
	}
	if j.cursor != 9 {
		t.Fatal("receive cursor was not committed")
	}
}
