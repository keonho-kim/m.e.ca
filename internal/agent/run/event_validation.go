// Purpose: Validate durable runtime event payloads before persistence.
// Pattern: Simple Module.
// Usage: Journal adapters use Event.Check at their write boundary.
// Related: internal/agent/run/events.go.
package run

import (
	"encoding/json"
	"fmt"
	modelapi "github.com/keonho-kim/v-code/internal/agent/model"
)

func (event Event) Check() error {
	if event.Monitor != nil {
		if err := event.Monitor.Check(); err != nil {
			return err
		}
	}
	if event.Conclave != nil {
		if err := event.Conclave.Check(); err != nil {
			return err
		}
	}
	if event.Grounding != nil {
		if err := event.Grounding.Check(); err != nil {
			return err
		}
	}
	if event.Kind == "usage" {
		if event.Usage == nil {
			return fmt.Errorf("missing model token usage")
		}
		if err := event.Usage.Check(); err != nil {
			return err
		}
	} else if event.Usage != nil {
		return fmt.Errorf("only final model usage may be persisted")
	}
	if event.Call != nil {
		if err := event.Call.Check(); err != nil {
			return err
		}
		if event.Call.ID != event.CallID {
			return fmt.Errorf("tool event call ID mismatch")
		}
	}
	if len(event.Output) > modelapi.MaxResponseBytes || len(event.Output) > 0 && !json.Valid(event.Output) {
		return fmt.Errorf("invalid tool event output")
	}
	if event.Checkpoint != nil {
		if err := event.Checkpoint.Check(); err != nil {
			return err
		}
	}
	return nil
}
