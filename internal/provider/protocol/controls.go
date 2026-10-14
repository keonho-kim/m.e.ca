// Purpose: Translate validated ON/OFF and model-specific effort choices into wire controls.
// Pattern: Provider Adapter.
// Usage: Generate supplies one request to controls before any HTTP call.
// Related: internal/provider/profile/effort.go, internal/provider/protocol/request.go, internal/provider/protocol/wire.go.
package protocol

import (
	"fmt"
	modelapi "github.com/keonho-kim/v-code/internal/agent/model"
	modelconn "github.com/keonho-kim/v-code/internal/provider/connection"
	"github.com/keonho-kim/v-code/internal/provider/profile"
)

type Controls struct {
	Think           any
	Template        map[string]any
	ReasoningEffort string
}

func ControlsFor(kind modelconn.ProviderKind, profileID string, info *modelconn.ModelInfo, request modelapi.Request) (Controls, error) {
	controls := Controls{}
	if info != nil && info.ID != request.ModelID {
		return controls, fmt.Errorf("request model differs from prepared model")
	}
	if info != nil && info.Reasoning != nil {
		return nativeControls(kind, info.Reasoning, request)
	}
	selected, err := profile.Resolve(request.ModelID, profileID)
	if err != nil {
		return controls, err
	}
	if err = selected.CheckEffort(kind, request.ReasoningEffort); err != nil {
		return controls, err
	}
	if kind == modelconn.Ollama {
		controls.Think = request.Thinking
		return controls, nil
	}
	keys := selected.ThinkingKeys(kind)
	if request.Thinking != nil && len(keys) == 0 {
		return controls, fmt.Errorf("Thinking ON/OFF control is unverified for this provider")
	}
	if request.Thinking != nil {
		controls.Template = map[string]any{}
		for _, key := range keys {
			controls.Template[key] = *request.Thinking
		}
	}
	if request.ReasoningEffort == "" {
		return controls, nil
	}
	if request.Thinking == nil {
		return controls, fmt.Errorf("an explicit Thinking choice is required for effort control")
	}
	if !*request.Thinking {
		return controls, nil
	}
	switch selected.Effort.Kind {
	case profile.ReasoningEffort:
		if kind == modelconn.SGLang {
			// SGLang's top-level normalization only toggles thinking. Pass the model
			// template's actual string input explicitly to retain its effort semantics.
			controls.Template["reasoning_effort"] = request.ReasoningEffort
		} else {
			controls.ReasoningEffort = request.ReasoningEffort
		}
	case profile.LowEffort:
		controls.Template["low_effort"] = request.ReasoningEffort == "low"
	default:
		return controls, fmt.Errorf("unknown profile effort control")
	}
	return controls, nil
}
