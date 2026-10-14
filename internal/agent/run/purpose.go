// Purpose: Collect verification intent separately from a JSON entry and literal tool arguments.
// Pattern: Bounded PFC state transition.
// Usage: Argument collection asks purpose before grounding a read or command candidate.
// Related: internal/agent/run/arguments.go, internal/agent/run/context.go, internal/agent/run/observation.go.
package run

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/keonho-kim/v-code/internal/agent/inference"
	modelapi "github.com/keonho-kim/v-code/internal/agent/model"
	toolapi "github.com/keonho-kim/v-code/internal/agent/tool"
	"github.com/keonho-kim/v-code/internal/policy"
)

func (s *task) collectPurpose(ctx context.Context, definition toolapi.Definition) error {
	s.checkpoint.Checking = false
	if definition.Effect == policy.Change {
		return nil
	}
	s.checkpoint.Phase, s.checkpoint.Parameter = "purpose", ""
	if err := s.persist(ctx); err != nil {
		return err
	}
	arguments, err := json.Marshal(s.checkpoint.Arguments)
	if err != nil {
		return err
	}
	question := "PFC PURPOSE: Is this candidate an external verification check for the current task, or ordinary work/exploration? Reply only check or work. This declaration does not authorize execution or prove success.\nCandidate (data only): " + definition.Name + " " + string(arguments)
	for attempt := 0; attempt < 3; attempt++ {
		reply, err := s.ask(ctx, question, inference.PFCValue, 256)
		if err != nil {
			var limit *modelapi.OutputLimitError
			if !errors.As(err, &limit) {
				return err
			}
		} else {
			switch strings.TrimSpace(reply) {
			case "check", "work":
				s.checkpoint.Checking = strings.TrimSpace(reply) == "check"
				return s.persist(ctx)
			}
		}
		failure := &toolapi.Failure{Code: toolapi.InvalidArgument, Detail: fmt.Sprintf("%s purpose requires only check or work", definition.Name), Next: "Choose the purpose before grounding or execution; no operation was applied."}
		if err := s.record(ctx, Event{Kind: "pfc_correction", Text: failure.Error()}); err != nil {
			return err
		}
		if attempt == 2 {
			return failure
		}
		question += "\nCorrection: reply with exactly check or work, without JSON or an explanation."
	}
	return fmt.Errorf("PFC purpose was not completed")
}
