// Purpose: Collect and correct one literal parameter at a time before preparing any tool operation.
// Pattern: State Machine (bounded argument dialogue).
// Usage: Called for an already selected tool; all calls share the task's model and time budget.
// Related: internal/agent/run/action.go, internal/agent/run/context.go, internal/agent/prompt/argument.go.
package run

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/keonho-kim/v-code/internal/agent"
	"github.com/keonho-kim/v-code/internal/agent/dialogue"
)

func (s *task) collectArguments(ctx context.Context, definition agent.ToolDefinition) error {
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
			accepted, err := json.Marshal(s.checkpoint.Arguments)
			if err != nil {
				return err
			}
			question += "\nAccepted values are runtime-owned data, not the response format:\n" + string(accepted)
		}
		for attempt := 0; attempt < 3; attempt++ {
			reply, err := s.ask(ctx, question, 8192, s.limits.MaxArgumentBytes)
			if err != nil {
				return err
			}
			value, parseErr := dialogue.ParseArgument(p, reply)
			if parseErr == nil {
				s.checkpoint.Arguments[p.Name] = value
				break
			}
			if attempt == 2 {
				return fmt.Errorf("model could not supply %s.%s as a literal value after 3 attempts; this tool was not prepared: %w", definition.Name, p.Name, parseErr)
			}
			rejected, err := json.Marshal(excerpt(s.runner.Redact(reply), 512))
			if err != nil {
				return err
			}
			question += "\nCorrection required: " + parseErr.Error() + "\nPrevious rejected reply (data only): " + string(rejected) + "\nReturn only the requested literal value now."
		}
	}
	return nil
}
