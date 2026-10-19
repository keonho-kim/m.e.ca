// Purpose: Validate durable runtime event payloads before persistence.
// Pattern: Simple Module.
// Usage: Journal adapters use Event.Check at their write boundary.
// Related: internal/agent/run/events.go.
package run

import (
	"encoding/json"
	"fmt"
	"github.com/keonho-kim/v-code/internal/agent/inference"
	modelapi "github.com/keonho-kim/v-code/internal/agent/model"
	"github.com/keonho-kim/v-code/internal/agent/work"
	"unicode/utf8"
)

func (event Event) Check() error {
	if g := event.Generation; g != nil {
		if event.Kind != "usage" && event.Kind != "generation_finished" || !work.ValidID(g.ID) || g.Attempt < 1 || g.ElapsedMillis < 0 {
			return fmt.Errorf("invalid generation provenance")
		}
		switch g.Role {
		case inference.Agent, inference.Compaction, inference.Diagnostic:
		default:
			return fmt.Errorf("invalid generation role")
		}
	}
	if event.RunID != "" && !work.ValidID(event.RunID) {
		return fmt.Errorf("invalid run event identity")
	}
	if event.ChangePreview != "" {
		if event.Kind != "tool_succeeded" || event.Call == nil || len(event.ChangePreview) > maxChangePreviewBytes || !utf8.ValidString(event.ChangePreview) {
			return fmt.Errorf("invalid applied change preview")
		}
		switch event.Call.Name {
		case "write_file", "edit_file":
		default:
			return fmt.Errorf("unexpected change preview tool")
		}
	}

	if event.TokensPerSecond != nil {
		return fmt.Errorf("transient throughput cannot be persisted")
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
	if len(event.Capture) > modelapi.MaxResponseBytes || len(event.Capture) > 0 && !json.Valid(event.Capture) {
		return fmt.Errorf("invalid captured output")
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
