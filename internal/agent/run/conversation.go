// Purpose: Save request and recovery context before decision work and settle all owned work on exit.
// Pattern: Request lifecycle boundary.
// Usage: Runner.Run starts the conversation and defers close after successful resource assembly.
// Related: internal/agent/run/start.go, internal/agent/run/settle.go, internal/agent/run/commands.go.
package run

import (
	"context"
	"encoding/json"
	"errors"

	modelapi "github.com/keonho-kim/meca/internal/agent/model"
	runrecord "github.com/keonho-kim/meca/internal/agent/record"
	toolapi "github.com/keonho-kim/meca/internal/agent/tool"
)

func (s *execution) startConversation(ctx context.Context, input string) error {
	r := s.runner
	if commands, ok := r.Tools.(toolapi.CommandExecutor); ok {
		commands.BeginCommands(ctx, s.usage.deadline)
	}
	if previous := r.PreviousRun; previous != nil && (previous.Status == runrecord.Interrupted || previous.Status == runrecord.Stopped) {
		note := "Previous request: " + previous.Objective + "\nLast observed outcome: " + previous.Observation
		if previous.Blocker != "" {
			note += "\nObserved blocker: " + previous.Blocker
		}
		if err := s.save(ctx, modelapi.Message{Role: "system", Content: note}); err != nil {
			return err
		}
	}
	if !r.Resume {
		if err := s.save(ctx, modelapi.Message{Role: "user", Content: input, Images: r.Images}); err != nil {
			return err
		}
	}
	s.runner.Images = nil
	for _, doc := range r.SkillReads {
		s.allowExplicitSkill(doc.ID)
		body, err := json.Marshal(doc)
		if err != nil {
			return err
		}
		if err := s.saveSkillRead(ctx, doc, string(body)); err != nil {
			return err
		}
	}
	return s.readMentions(ctx, input)
}

func (s *execution) close(ctx context.Context, cause error) error {
	cause = errors.Join(cause, s.stopCommands(ctx))
	return s.settle(ctx, cause)
}
