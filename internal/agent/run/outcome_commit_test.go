// Purpose: Require terminal tool facts and their native replies to cross one durability boundary.
// Pattern: Journal boundary regression tests.
// Usage: go test ./internal/agent/run -run TestOutcomeCommit.
// Related: internal/agent/run/execution_result.go, internal/agent/run/action_result.go.
package run

import (
	"context"
	"errors"
	"testing"

	modelapi "github.com/keonho-kim/meca/internal/agent/model"
	toolapi "github.com/keonho-kim/meca/internal/agent/tool"
)

type outcomeJournal struct {
	*journal
	failReply bool
	t         *testing.T
}

func (j *outcomeJournal) Commit(ctx context.Context, session string, records []JournalRecord) error {
	for _, record := range records {
		if record.Event == nil || record.Event.Kind != "tool_succeeded" {
			continue
		}
		paired, checkpoint := false, false
		for _, sibling := range records {
			paired = paired || sibling.Message != nil && sibling.Message.ToolCallID == record.Event.CallID
			checkpoint = checkpoint || sibling.Event != nil && sibling.Event.Checkpoint != nil
		}
		if !paired || !checkpoint {
			j.t.Error("terminal outcome committed without its reply and checkpoint")
		}
	}
	if j.failReply {
		for _, record := range records {
			if record.Message != nil && record.Message.Role == "tool" {
				return errors.New("reply commit failed")
			}
		}
	}
	return j.journal.Commit(ctx, session, records)
}

func TestOutcomeCommitIsAtomic(t *testing.T) {
	for _, fail := range []bool{false, true} {
		j := &outcomeJournal{journal: &journal{}, failReply: fail, t: t}
		g := &generator{replies: []string{nativeCall("write_file", map[string]any{"path": "a", "content": "body"}), "done"}}
		r := fixture(g, &executor{}, j.journal)
		r.Journal = j
		_, err := r.Run(context.Background(), "write a", approve, emit)
		if (err != nil) != fail {
			t.Fatalf("failure=%v: %v", fail, err)
		}
		for _, event := range j.events {
			if fail && event.Kind == "tool_succeeded" {
				t.Fatal("outcome survived a rolled-back reply")
			}
		}
		if !fail {
			var reply *modelapi.Message
			for i := range j.messages {
				if j.messages[i].Role == "tool" {
					reply = &j.messages[i]
				}
			}
			if reply == nil || reply.ToolCallID == "" {
				t.Fatal("paired native result lost")
			}
		}
	}
}

func TestOutcomeCommitPrecedesDisplay(t *testing.T) {
	j := &journal{}
	g := &generator{replies: []string{nativeCall("write_file", map[string]any{"path": "a", "content": "body"}), "done"}}
	r := fixture(g, &executor{}, j)
	_, err := r.Run(context.Background(), "write a", approve, func(event Event) error {
		if event.Kind == "tool_succeeded" {
			for _, m := range j.messages {
				if m.Role == "tool" && m.ToolCallID == event.CallID {
					return nil
				}
			}
			t.Error("published an outcome before its native reply was durable")
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}

func TestOutcomeCommitDenialPreservesStorageFailure(t *testing.T) {
	j := &journal{failKind: "tool_failed"}
	g := &generator{replies: []string{nativeCall("write_file", map[string]any{"path": "a", "content": "body"})}}
	r := fixture(g, &executor{approval: true}, j)
	_, err := r.Run(context.Background(), "write a", func(context.Context, toolapi.Approval) (bool, error) { return false, nil }, emit)
	if !errors.Is(err, errRuntimeRecord) || !errors.Is(err, errApprovalDenied) {
		t.Fatalf("denial hid a failed outcome commit: %v", err)
	}
}
