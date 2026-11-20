// Purpose: Verify a token-limited answer continues only within a measured follow-up budget.
// Pattern: Prepared generator integration test.
// Usage: go test ./internal/agent/run -run TestBoundedTextContinuation.
// Related: internal/agent/run/decision_request.go, internal/agent/run/generate.go.
package run

import (
	"context"
	"testing"

	"github.com/keonho-kim/meca/internal/agent/inference"
	modelapi "github.com/keonho-kim/meca/internal/agent/model"
)

type continuationModel struct {
	requests, sends                                  int
	unmeasured, fullWindow, secondLimit, partialTool bool
	suffix                                           string
}

func (m *continuationModel) Generate(context.Context, modelapi.Request, func(modelapi.Event) error) error {
	panic("unprepared request")
}
func (m *continuationModel) Prepare(_ context.Context, r modelapi.Request) (modelapi.PreparedGeneration, error) {
	m.requests++
	budget := modelapi.InputBudget{Window: 4096, InputTokens: 100, OutputTokens: 300}
	if m.requests > 1 && m.unmeasured {
		budget.InputTokens = 0
	}
	if m.requests > 1 && m.fullWindow {
		budget.InputTokens = budget.Window - 1
	}
	return modelapi.PreparedGeneration{Budget: budget, Send: func(ctx context.Context, emit func(modelapi.Event) error) error {
		m.sends++
		if err := emit(modelapi.Event{InputBudget: &budget}); err != nil {
			return err
		}
		if m.sends == 1 {
			if err := emit(modelapi.Event{Delta: "First.", Streaming: true}); err != nil {
				return err
			}
			return &modelapi.OutputLimitError{Reason: "output token limit", TokenLimit: true, PartialTool: m.partialTool}
		}
		draft := r.Messages[len(r.Messages)-2]
		if len(r.Tools) != 0 || draft.Role != "assistant" || draft.Content != "First." || r.Messages[len(r.Messages)-1].Role != "user" {
			panic("continuation omitted the draft or offered tools")
		}
		if m.secondLimit {
			if err := emit(modelapi.Event{Delta: m.suffix, Streaming: true}); err != nil {
				return err
			}
			return &modelapi.OutputLimitError{Reason: "second limit", TokenLimit: true}
		}
		return emit(modelapi.Event{Delta: m.suffix, Done: true, Streaming: true, ReasoningState: []byte("suffix-only")})
	}}, nil
}
func TestBoundedTextContinuation(t *testing.T) {
	for _, tc := range []struct {
		name       string
		model      continuationModel
		explicit   bool
		sends      int
		want       string
		incomplete bool
	}{
		{"measured suffix", continuationModel{suffix: " Next."}, false, 2, "First. Next.", false},
		{"full replay", continuationModel{suffix: "First. Next."}, false, 2, "First. Next.", false},
		{"legitimate repeated prefix", continuationModel{suffix: "Fir"}, false, 2, "First.Fir", false},
		{"repeated token limit", continuationModel{suffix: " More", secondLimit: true}, false, 2, "First. More", true},
		{"no measured follow-up", continuationModel{unmeasured: true}, false, 1, "First.", true},
		{"insufficient room", continuationModel{fullWindow: true}, false, 1, "First.", true},
		{"user output cap", continuationModel{}, true, 1, "First.", true},
		{"partial native call", continuationModel{partialTool: true}, false, 1, "First.", true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			m := &tc.model
			j, executor := &journal{}, &executor{}
			r := fixture(&generator{}, executor, j)
			r.Generator = m
			if tc.explicit {
				settings := inference.Defaults()
				settings.Agent.MaxOutputTokens = 12
				r.Inference = &settings
			}
			result, err := r.Run(t.Context(), "Continue.", approve, emit)
			if m.sends != tc.sends || (err != nil) != tc.incomplete || executor.executed != 0 {
				t.Fatalf("sends=%d result=%q err=%v executed=%d", m.sends, result.Text, err, executor.executed)
			}
			var answers []modelapi.Message
			for _, msg := range j.messages {
				if msg.Role == "assistant" {
					answers = append(answers, msg)
				}
			}
			if len(answers) != 1 || answers[0].Content != tc.want || answers[0].Incomplete != tc.incomplete || len(answers[0].ReasoningState) != 0 {
				t.Fatalf("stored answers=%+v", answers)
			}
		})
	}
}

func TestNewInstructionSupersedesPreparedContinuation(t *testing.T) {
	r := fixture(&generator{}, &executor{}, &journal{})
	pending := false
	r.PollSteering = func() []string {
		if !pending {
			return nil
		}
		pending = false
		return []string{"Use the updated request"}
	}
	preparations, sends := 0, 0
	r.Generator = preparingModel{prepare: func(_ context.Context, request modelapi.Request) (modelapi.PreparedGeneration, error) {
		preparations++
		if preparations == 2 {
			pending = true
		}
		return modelapi.PreparedGeneration{Budget: modelapi.InputBudget{Window: 1024, InputTokens: 100, OutputTokens: 200}, Send: func(_ context.Context, emit func(modelapi.Event) error) error {
			sends++
			if sends == 1 {
				if err := emit(modelapi.Event{Delta: "Stale draft", Streaming: true}); err != nil {
					return err
				}
				return &modelapi.OutputLimitError{Reason: "limit", TokenLimit: true}
			}
			if len(request.Tools) == 0 || request.Messages[len(request.Messages)-1].Content != "Use the updated request" {
				t.Fatal("sent stale continuation instead of admitting the updated task")
			}
			return emit(modelapi.Event{Delta: "Updated answer", Done: true})
		}}, nil
	}}
	result, err := r.Run(t.Context(), "Original request", approve, emit)
	if err != nil || result.Text != "Updated answer" || sends != 2 || preparations != 3 {
		t.Fatalf("reply=%q error=%v sends=%d preparations=%d", result.Text, err, sends, preparations)
	}
}
