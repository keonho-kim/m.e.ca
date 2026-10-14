// Purpose: Validate model selection, identity binding and reasoning choices independently of storage.
// Pattern: Simple Module.
// Usage: Storage adapters validate selections on both write and read.
// Related: internal/provider/connection/models.go, internal/provider/profile/effort.go.
package selection

import (
	"fmt"
	modelconn "github.com/keonho-kim/v-code/internal/provider/connection"
	"github.com/keonho-kim/v-code/internal/provider/profile"
)

func Validate(selection modelconn.Selection) error {
	parsed, err := modelconn.New(selection.Connection.Kind, selection.Connection.Endpoint)
	if err != nil {
		return err
	}
	if parsed.ID != selection.Connection.ID || selection.Model == "" || len(selection.Model) > 512 {
		return fmt.Errorf("invalid model selection")
	}
	if selection.Identity != nil {
		if err := selection.Identity.Validate(); err != nil {
			return err
		}
	}
	selected, err := profile.Resolve(selection.Model, selection.Profile)
	if err != nil {
		return err
	}
	if err = checkEffort(selection, selected); err != nil {
		return err
	}
	return nil
}

func checkEffort(selected modelconn.Selection, resolved profile.Profile) error {
	if selected.Details != nil {
		if selected.Details.ID != selected.Model {
			return fmt.Errorf("model metadata binding mismatch")
		}
		if err := selected.Details.Check(); err != nil {
			return err
		}
		if control := selected.Details.Reasoning; control != nil {
			if selected.ThinkingSupported && (!control.Enabled() || !control.Off || selected.Thinking == nil) {
				return fmt.Errorf("invalid native ON/OFF capability")
			}
			if selected.Effort() != "" && selected.Thinking == nil {
				return fmt.Errorf("native effort has no verified Thinking setting")
			}
			if selected.Connection.Kind != modelconn.Ollama && selected.Connection.Kind != modelconn.LMStudio {
				return fmt.Errorf("native reasoning metadata on incompatible provider")
			}
			if selected.Thinking != nil && ((!*selected.Thinking && !control.Off) || (*selected.Thinking && !control.Enabled())) {
				return fmt.Errorf("unsupported native Thinking choice")
			}
			return control.CheckEffort(selected.Effort())
		}
	}
	return resolved.CheckEffort(selected.Connection.Kind, selected.Effort())
}
