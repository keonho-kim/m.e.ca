// Purpose: Apply new user instructions between completed model/tool turns.
// Pattern: Request-owned input boundary.
// Usage: The loop polls its UI-independent queue only after workers have joined.
// Related: internal/agent/run/run.go, internal/tui/session/steering.go.
package run

import (
	"context"
	"fmt"
	modelapi "github.com/keonho-kim/v-code/internal/agent/model"
)

func (s *execution) steer(ctx context.Context) (bool, error) {
	if s.runner.PollSteering == nil {
		return false, nil
	}
	messages := s.runner.PollSteering()
	return len(messages) > 0, s.applyInstructions(ctx, messages)
}

func (s *execution) applyInstructions(ctx context.Context, messages []string) error {
	for _, text := range messages {
		if len(text) > 64*1024*1024 {
			return fmt.Errorf("user instruction exceeds 64 MiB")
		}
		if err := s.save(ctx, modelapi.Message{Role: "user", Content: text}); err != nil {
			return err
		}
		s.checkpoint.Objective = s.runner.Redact(text)
		if err := s.emit(Event{Kind: "user_message", Text: s.runner.Redact(text)}); err != nil {
			return err
		}
	}
	return nil
}

func (s *execution) continueSteered(ctx context.Context, reply modelapi.Response) (bool, error) {
	if s.runner.PollSteering == nil {
		return false, nil
	}
	messages := s.runner.PollSteering()
	if len(messages) == 0 {
		return false, nil
	}
	if reply.Text != "" {
		if err := s.save(ctx, modelapi.Message{Role: "assistant", Content: reply.Text, Reasoning: reply.Reasoning}); err != nil {
			return false, err
		}
		if err := s.emit(Event{Kind: "progress", Text: reply.Text}); err != nil {
			return false, err
		}
	}
	return true, s.applyInstructions(ctx, messages)
}
