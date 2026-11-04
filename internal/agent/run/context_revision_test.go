// Purpose: Invalidate generated effects on every new user-message sequence, including repeated text.
// Pattern: Scripted generation-boundary regression test.
// Usage: go test ./internal/agent/run -run TestRepeatedObjective.
// Related: internal/agent/run/context_inbox.go, internal/agent/run/loop.go.
package run

import (
	"context"
	"encoding/json"
	model "github.com/keonho-kim/meca/internal/agent/model"
	"github.com/keonho-kim/meca/internal/agent/tool"
	"strings"
	"testing"
)

type contextDecisionTools struct {
	*executor
	check func() error
}

func (e contextDecisionTools) CheckUserDecision() error { return e.check() }

func TestRepeatedObjectiveDoesNotHideInterveningUserRestriction(t *testing.T) {
	g := &generator{replies: []string{nativeCall("write_file", map[string]any{"path": "auth.py", "content": "obsolete candidate"}), "Understood the current restriction"}}
	j := &contextJournalFixture{}
	tools := &executor{}
	r := fixture(g, tools, &j.journal)
	r.Journal = j
	r.PollContext = func(context.Context) (ContextBatch, error) {
		if len(g.requests) == 1 && j.cursor == 0 {
			return ContextBatch{Cursor: 2, UserSequence: 2, Objective: "Continue implementation", Reconsider: true, Messages: []model.Message{{Role: "user", Content: "Do not modify auth.py."}, {Role: "user", Content: "Continue implementation"}}}, nil
		}
		return ContextBatch{}, nil
	}
	_, err := r.Run(context.Background(), "Continue implementation", approve, emit)
	if err != nil || tools.prepared != 0 || tools.executed != 0 {
		t.Fatal("new user-message identity did not invalidate the old candidate", tools.executed, err)
	}
}

func TestUserInstructionBetweenWavesPreservesOnlyCompletedEffects(t *testing.T) {
	j := &contextJournalFixture{}
	tools := &executor{}
	r := fixture(&generator{}, tools, &j.journal)
	r.Journal = j
	r.Tools = contextDecisionTools{tools, func() error {
		if tools.executed == 1 && j.cursor == 0 {
			return tool.NewInstructionChanged()
		}
		return nil
	}}
	requests := 0
	sawInstructionFeedback := false
	r.Generator = boundaryModel(func(_ context.Context, request model.Request, emit func(model.Event) error) error {
		requests++
		if requests > 1 {
			for _, message := range request.Messages {
				sawInstructionFeedback = sawInstructionFeedback || strings.Contains(message.Content, tool.InstructionChanged) && strings.Contains(message.Content, "latest user")
			}
			return emit(model.Event{Delta: "Preserved the second file", Done: true})
		}
		var calls []tool.Call
		for _, path := range []string{"first.py", "second.py"} {
			args, _ := json.Marshal(map[string]string{"path": path, "content": "content"})
			calls = append(calls, tool.Call{ID: path, Name: "write_file", Arguments: args})
		}
		return emit(model.Event{Calls: calls, Done: true})
	})
	r.PollContext = func(context.Context) (ContextBatch, error) {
		if tools.executed == 1 && j.cursor == 0 {
			return ContextBatch{Cursor: 2, UserSequence: 2, Objective: "Build files", Messages: []model.Message{{Role: "user", Content: "Keep second.py unchanged."}, {Role: "user", Content: "Build files"}}}, nil
		}
		return ContextBatch{}, nil
	}
	_, err := r.Run(context.Background(), "Build files", approve, emit)
	if err != nil || tools.prepared != 1 || tools.executed != 1 || requests != 2 {
		t.Fatal("a later wave ignored steering or replayed the completed operation", tools.executed, err)
	}
	if !sawInstructionFeedback {
		t.Fatal("the next decision did not receive instruction-specific non-execution feedback")
	}
}
