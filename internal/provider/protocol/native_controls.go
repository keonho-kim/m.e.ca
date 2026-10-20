// Purpose: Encode reasoning choices using provider request fields and literal server levels.
// Pattern: Provider wire adapter.
// Usage: Prepared clients call nativeControls while encoding generation requests.
// Related: internal/provider/connection/reasoning.go, internal/provider/protocol/controls.go.
package protocol

import (
	"fmt"
	modelapi "github.com/keonho-kim/v-code/internal/agent/model"
	"github.com/keonho-kim/v-code/internal/provider/connection"
)

func nativeControls(kind connection.ProviderKind, r *connection.Reasoning, request modelapi.Request) (Controls, error) {
	controls := Controls{}
	if err := r.CheckEffort(request.ReasoningEffort); err != nil {
		return controls, err
	}
	if request.Thinking == nil {
		if request.ReasoningEffort != "" {
			return controls, fmt.Errorf("explicit Thinking is required for effort")
		}
		return controls, nil
	}
	if kind == connection.VLLM || kind == connection.SGLang {
		return templateControls(kind, r, request)
	}
	if !*request.Thinking {
		if !r.Off {
			return controls, fmt.Errorf("this model does not advertise Thinking OFF")
		}
		if kind == connection.Ollama {
			controls.Think = false
		} else {
			controls.ReasoningEffort = "none"
		}
		return controls, nil
	}
	if !r.Enabled() {
		return controls, fmt.Errorf("this model does not advertise Thinking ON")
	}
	level := request.ReasoningEffort
	if level == "" {
		level = r.Default
		if level == "" && !r.On {
			level = r.Levels[0]
		}
	}
	if kind == connection.Ollama {
		controls.Think = true
		if level != "" {
			controls.Think = level
		}
	} else {
		controls.ReasoningEffort = level
		// Responses uses the OpenAI effort vocabulary. Binary ON is probed using
		// medium; it does not expose additional strength choices in the menu.
		if level == "" {
			controls.ReasoningEffort = "medium"
		}
	}
	return controls, nil
}

func templateControls(kind connection.ProviderKind, r *connection.Reasoning, request modelapi.Request) (Controls, error) {
	on := *request.Thinking
	if on && !r.Enabled() || !on && !r.Off {
		return Controls{}, fmt.Errorf("Thinking choice is unavailable for this connection")
	}
	controls := Controls{Template: map[string]any{"enable_thinking": on, "thinking": on}}
	if on && request.ReasoningEffort != "" {
		if kind == connection.SGLang {
			controls.Template["reasoning_effort"] = request.ReasoningEffort
		} else {
			controls.ReasoningEffort = request.ReasoningEffort
		}
	}
	return controls, nil
}
