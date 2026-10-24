// Purpose: Verify resuming an interrupted generation retains executed work without replaying thoughts or effects.
// Pattern: Runtime lifecycle contract tests.
// Usage: go test ./internal/agent/run -run TestInterruptedGeneration.
// Related: internal/agent/run/generate.go, internal/agent/run/action.go, internal/agent/context/conversation.go.
package run

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"testing"

	modelapi "github.com/keonho-kim/meca/internal/agent/model"
	toolapi "github.com/keonho-kim/meca/internal/agent/tool"
)

type interruptedGenerator struct {
	mode   string
	calls  int
	cancel context.CancelFunc
}

func (g *interruptedGenerator) Generate(ctx context.Context, _ modelapi.Request, emit func(modelapi.Event) error) error {
	g.calls++
	if g.calls == 1 {
		return emit(modelapi.Event{Done: true, Thinking: "completed tool reasoning", Calls: []toolapi.Call{{ID: "created", Name: "write_file", Arguments: json.RawMessage(`{"path":"README.md","content":"# Project"}`)}}})
	}
	if err := emit(modelapi.Event{Streaming: true, Thinking: "unfinished secret thought", Delta: "All work is finished"}); err != nil {
		return err
	}
	switch g.mode {
	case "cancel":
		g.cancel()
		return ctx.Err()
	case "disconnect":
		return fmt.Errorf("stream disconnected")
	default:
		return nil // EOF without a completion event is also an unfinished reply.
	}
}

func TestInterruptedGenerationRetainsExecutedWork(t *testing.T) {
	for _, mode := range []string{"cancel", "disconnect", "missing_done"} {
		t.Run(mode, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			j, e := &journal{}, &executor{}
			r := fixture(&generator{}, e, j)
			r.Generator = &interruptedGenerator{mode: mode, cancel: cancel}
			first, err := r.Run(ctx, "Create README.md", approve, emit)
			if err == nil || e.executed != 1 || mode == "cancel" && !errors.Is(err, context.Canceled) {
				t.Fatalf("initial outcome: status=%s executed=%d err=%v", first.Status, e.executed, err)
			}
			var thoughtSaved bool
			for _, event := range j.events {
				thoughtSaved = thoughtSaved || event.Kind == "thinking" && event.Thinking == "unfinished secret thought"
				if event.Checkpoint != nil {
					r.PreviousRun = event.Checkpoint
				}
			}
			if !thoughtSaved || !j.messages[len(j.messages)-1].Incomplete {
				t.Fatal("interrupted output not archived")
			}
			g := &generator{replies: []string{"README.md was created; its contents remain unchecked."}}
			r.Generator = g
			result, err := r.Run(context.Background(), "Report current progress", approve, emit)
			if err != nil || result.Status != Answered || e.executed != 1 {
				t.Fatalf("resume replayed effects or failed: %+v executed=%d err=%v", result, e.executed, err)
			}
			messages := g.requests[0].Messages
			body, _ := json.Marshal(messages)
			if strings.Contains(string(body), "unfinished secret thought") || !strings.Contains(string(body), "Unfinished assistant draft") || !strings.Contains(string(body), "Create README.md") {
				t.Fatalf("incorrect continuation context: %s", body)
			}
			paired := false
			for i, message := range messages {
				if len(message.Calls) == 0 {
					continue
				}
				paired = message.Role == "assistant" && message.Reasoning == "completed tool reasoning" && i+1 < len(messages) && messages[i+1].Role == "tool" && messages[i+1].ToolCallID == message.Calls[0].ID && strings.Contains(messages[i+1].Content, "changed")
			}
			if !paired || messages[len(messages)-1].Content != "Report current progress" {
				t.Fatal("completed tool reasoning, result pairing or latest instruction lost")
			}
		})
	}
}
