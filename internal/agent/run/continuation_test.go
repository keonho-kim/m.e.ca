// Purpose: Verify opaque provider state survives tool execution and later user requests without entering UI events.
// Pattern: Scripted conversation contract tests.
// Usage: go test ./internal/agent/run -run TestProviderContinuation.
// Related: internal/agent/run/generate.go, internal/agent/context/conversation.go, internal/agent/run/finish.go.
package run

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	modelapi "github.com/keonho-kim/meca/internal/agent/model"
	runrecord "github.com/keonho-kim/meca/internal/agent/record"
	toolapi "github.com/keonho-kim/meca/internal/agent/tool"
)

type continuationGenerator struct {
	t     *testing.T
	step  int
	calls []toolapi.Call
}

func (g *continuationGenerator) Generate(_ context.Context, request modelapi.Request, emit func(modelapi.Event) error) error {
	g.step++
	if g.step == 1 {
		return emit(modelapi.Event{Delta: "Inspecting", Thinking: "brief", Calls: g.calls, ReasoningState: []byte("tool-state"), Done: true})
	}
	want := "tool-state"
	if g.step == 3 {
		want = "answer-state"
	}
	found := false
	for _, message := range request.Messages {
		if string(message.ReasoningState) == want {
			found = true
			if message.Role != "assistant" || message.ProviderID != "provider" || message.ModelID != "model" {
				g.t.Errorf("continuation lost its owner: %+v", message)
			}
		}
	}
	if !found {
		g.t.Errorf("missing %s in request %d", want, g.step)
	}
	return emit(modelapi.Event{Delta: "Done", Thinking: "final thought", ReasoningState: []byte("answer-state"), Done: true})
}

func TestProviderContinuationSurvivesToolAndFinalReplies(t *testing.T) {
	for _, count := range []int{1, 2} {
		t.Run(fmt.Sprint(count), func(t *testing.T) {
			g := &continuationGenerator{t: t}
			for i := 0; i < count; i++ {
				g.calls = append(g.calls, toolapi.Call{ID: fmt.Sprintf("call_%d", i), Name: "inspect_fixture", Arguments: json.RawMessage(`{"path":"a"}`)})
			}
			j := &journal{}
			r := fixture(&generator{}, &executor{}, j)
			r.Generator, r.Provider, r.Model = g, "provider", "model"
			for _, input := range []string{"inspect files", "continue"} {
				result, err := r.Run(context.Background(), input, approve, func(event runrecord.Event) error {
					body, err := json.Marshal(event)
					if err != nil {
						return err
					}
					if strings.Contains(string(body), "tool-state") || strings.Contains(string(body), "answer-state") {
						t.Fatal("opaque state appeared in UI event")
					}
					return nil
				})
				if err != nil || result.Status != runrecord.Answered {
					t.Fatalf("result=%+v err=%v", result, err)
				}
			}
			last := j.messages[len(j.messages)-1]
			if string(last.ReasoningState) != "answer-state" || last.Reasoning != "final thought" {
				t.Fatalf("final continuation not persisted: %+v", last)
			}
		})
	}
}
