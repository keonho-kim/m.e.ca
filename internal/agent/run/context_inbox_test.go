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
	"github.com/keonho-kim/meca/internal/agent/team/channel"
)

type contextJournalFixture struct {
	journal
	cursor int64
}

func TestSanitizedPeerContextDetachesFromItsOriginalSharedBytes(t *testing.T) {
	j := &contextJournalFixture{}
	r := fixture(&generator{replies: []string{"done"}}, &executor{}, &j.journal)
	r.Journal = j
	r.Redact = func(text string) string { return strings.ReplaceAll(text, "provider-secret", "[redacted]") }
	r.PollContext = func(context.Context) (ContextBatch, error) {
		if j.cursor == 0 {
			return ContextBatch{Cursor: 9, Messages: []modelapi.Message{{ID: "peer:Ren/9", Role: "system", Content: "Public conversation — Ren\nprovider-secret"}}}, nil
		}
		return ContextBatch{}, nil
	}
	if _, err := r.Run(context.Background(), "Continue", approve, emit); err != nil {
		t.Fatal(err)
	}
	for _, message := range j.messages {
		if channel.IsPeer(message.ID) {
			if _, shared := channel.Sequence(message.ID); shared || strings.Contains(message.Content, "provider-secret") {
				t.Fatal("sanitized context kept a reference to different original bytes", message)
			}
			return
		}
	}
	t.Fatal("peer observation was not committed")
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
			return ContextBatch{Cursor: 1, UserSequence: 1, Objective: "Do not create the file", Messages: []modelapi.Message{{Role: "user", Content: "Do not create the file"}}}, nil
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
			return ContextBatch{Cursor: 9, Messages: []modelapi.Message{{ID: "peer:Ren/9", Role: "system", Content: "Public conversation — Ren\nUse the existing storage interface."}}}, nil
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

func TestContextCursorWithoutNewMessagesDoesNotRequestAnotherInference(t *testing.T) {
	g := &generator{replies: []string{"Finished the requested change"}}
	j := &contextJournalFixture{}
	r := fixture(g, &executor{}, &j.journal)
	r.Journal = j
	r.PollContext = func(context.Context) (ContextBatch, error) {
		if len(g.requests) == 1 && j.cursor == 0 {
			return ContextBatch{Cursor: 7}, nil
		}
		return ContextBatch{}, nil
	}
	result, err := r.Run(context.Background(), "Implement a change", approve, emit)
	if err != nil || result.Status != runrecord.Answered || len(g.requests) != 1 || j.cursor != 7 {
		t.Fatal("passive observation forced inference or lost its durable cursor", result, len(g.requests), err)
	}
}
