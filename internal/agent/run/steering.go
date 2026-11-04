// Purpose: Admit current user instructions at serial decision and execution boundaries.
// Pattern: Request-owned input boundary.
// Usage: The loop polls its UI-independent queue only after workers have joined.
// Related: internal/agent/run/run.go, internal/tui/session/steering.go.
package run

import (
	"context"
	"fmt"

	modelapi "github.com/keonho-kim/meca/internal/agent/model"
	runrecord "github.com/keonho-kim/meca/internal/agent/record"
	"github.com/keonho-kim/meca/internal/agent/tool"
)

// admitInstructions merges the mode-specific inputs on the serial runtime owner.
func (s *execution) admitInstructions(ctx context.Context) (bool, error) {
	changed, err := s.admitContext(ctx)
	if err != nil {
		return false, err
	}
	steered, err := s.steer(ctx)
	return changed || steered, err
}

func (s *execution) checkInstructions(ctx context.Context, revision uint64) error {
	if _, err := s.admitContext(ctx); err != nil {
		return err
	}
	return s.checkSteering(ctx, revision)
}

// instructionGuard checks cooperative execution without receiving conversation
// inside an outstanding native call/result group.
type instructionGuard interface{ CheckUserDecision() error }

func (s *execution) checkSteering(ctx context.Context, revision uint64) error {
	if _, err := s.steer(ctx); err != nil {
		return err
	}
	if revision != s.instructionRevision {
		return tool.NewInstructionChanged()
	}
	if guard, ok := s.runner.Tools.(instructionGuard); ok {
		return guard.CheckUserDecision()
	}
	return nil
}

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
		s.instructionRevision++
		s.checkpoint.Objective = s.runner.Redact(text)
		if err := s.emit(runrecord.Event{Kind: "user_message", Text: s.runner.Redact(text)}); err != nil {
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
		if err := s.save(ctx, modelapi.Message{Role: "assistant", Content: reply.Text, Reasoning: reply.Reasoning, ReasoningState: reply.ReasoningState}); err != nil {
			return false, err
		}
		if err := s.emit(runrecord.Event{Kind: "progress", Text: reply.Text}); err != nil {
			return false, err
		}
	}
	return true, s.applyInstructions(ctx, messages)
}
