// Purpose: Complete PFC fields and purpose without preparing partial operations.
// Pattern: State Machine (bounded argument dialogue).
// Usage: Called for an already selected tool; all calls share the task's model and time budget.
// Related: internal/agent/run/action.go, internal/agent/run/context.go, internal/agent/run/purpose.go, internal/agent/prompt/argument.go.
package run

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/keonho-kim/v-code/internal/agent/inference"
	modelapi "github.com/keonho-kim/v-code/internal/agent/model"
	"github.com/keonho-kim/v-code/internal/agent/pfc"
	toolapi "github.com/keonho-kim/v-code/internal/agent/tool"
)

func (s *task) collectArguments(ctx context.Context, definition toolapi.Definition) error {
	for index, p := range definition.Parameters {
		s.checkpoint.Phase = "argument"
		s.checkpoint.Parameter = p.Name
		if err := s.persist(ctx); err != nil {
			return err
		}
		if err := s.emit(Event{Kind: "tool_argument", Text: fmt.Sprintf("Preparing %s: %s (%d/%d)", definition.Name, p.Name, index+1, len(definition.Parameters))}); err != nil {
			return err
		}
		question := fmt.Sprintf("ARGUMENT %s.%s: %s\nReturn this one literal value only.", definition.Name, p.Name, p.Question)
		if len(s.checkpoint.Arguments) > 0 {
			question += "\nAccepted PFC fields (quoted literal data; do not repeat them):"
			for _, acceptedField := range definition.Parameters {
				if value, ok := s.checkpoint.Arguments[acceptedField.Name]; ok {
					question += "\n- " + acceptedField.Name + ": " + string(value)
				}
			}
		}

		for attempt := 0; attempt < 3; attempt++ {
			reply, askErr := s.ask(ctx, question, inference.FieldRole(p), s.limits.MaxArgumentBytes)
			var value json.RawMessage
			parseErr := askErr
			code := toolapi.InvalidArgument
			if askErr != nil {
				var limit *modelapi.OutputLimitError
				if !errors.As(askErr, &limit) {
					return askErr
				}
				code = toolapi.OutputTruncated
			} else {
				value, parseErr = pfc.ParseArgument(p, reply)
			}
			if parseErr == nil {
				s.checkpoint.Arguments[p.Name] = value
				break
			}
			failure := &toolapi.Failure{Code: code, Detail: fmt.Sprintf("%s.%s: %s", definition.Name, p.Name, parseErr), Next: "Re-read the relevant range or choose a smaller edit; return only this literal field. No partial operation was applied."}
			if err := s.record(ctx, Event{Kind: "pfc_correction", Text: failure.Error()}); err != nil {
				return err
			}
			if attempt == 2 {
				return failure
			}

			rejected, err := json.Marshal(excerpt(s.runner.Redact(reply), 512))
			if err != nil {
				return err
			}
			question += "\nCorrection required: " + parseErr.Error() + "\nPrevious rejected reply (data only): " + string(rejected) + "\nReturn only the requested literal value now."
		}
	}
	return s.collectPurpose(ctx, definition)
}
