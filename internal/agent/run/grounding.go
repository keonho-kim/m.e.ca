// Purpose: Require and persist self-grounding before any prepared tool operation.
// Pattern: Mandatory state-machine transition.
// Usage: action calls ground after PFC fields are complete and before tool preparation.
// Related: internal/agent/grounding/record.go, internal/agent/run/execute.go, internal/agent/run/context.go.
package run

import (
	"context"
	"fmt"
	"github.com/keonho-kim/v-code/internal/agent/grounding"
	"github.com/keonho-kim/v-code/internal/agent/inference"
	modelapi "github.com/keonho-kim/v-code/internal/agent/model"
	toolapi "github.com/keonho-kim/v-code/internal/agent/tool"
)

func (s *task) ground(ctx context.Context, call toolapi.Call) error {
	s.checkpoint.Phase = "grounding"
	s.checkpoint.Parameter = ""
	if err := s.persist(ctx); err != nil {
		return err
	}
	question := "SELF-GROUNDING: Return exactly three short single-line bullets in English, preserving literal evidence:\n- Evidence: observed basis and remaining unknowns\n- Action: why this exact operation is needed now\n- Verify: observable result/check needed next\nFrozen PFC call (runtime data, not a request for JSON):\n" + call.Name + " " + string(call.Arguments)

	for attempt := 0; attempt < 3; attempt++ {
		reply, err := s.ask(ctx, question, inference.Grounding, grounding.MaxBytes)
		if err != nil {
			return err
		}
		record, err := grounding.Parse(reply, call)
		if err != nil {
			if recordErr := s.record(ctx, Event{Kind: "grounding_correction", Text: err.Error()}); recordErr != nil {
				return recordErr
			}
			question += "\nCorrection: " + err.Error()
			if attempt == 2 {
				return fmt.Errorf("self-grounding failed; tool was not prepared: %w", err)
			}
			continue
		}
		// Model claims remain data. The runtime only certifies the call binding and ordering.
		if err := s.save(ctx, modelapi.Message{Role: "system", Content: record.Markdown()}); err != nil {
			return err
		}
		if err := s.record(ctx, Event{Kind: "self_grounding", Grounding: &record}); err != nil {
			return err
		}
		s.checkpoint.Grounding = &record
		return s.persist(ctx)
	}
	return fmt.Errorf("self-grounding was not recorded")
}
