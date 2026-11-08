// Purpose: Encode reasoning choices using provider request fields and literal server levels.
// Pattern: Provider wire adapter.
// Usage: Prepared clients call nativeControls while encoding generation requests.
// Related: internal/provider/connection/reasoning.go, internal/provider/protocol/controls.go.
package protocol

import (
	modelapi "github.com/keonho-kim/meca/internal/agent/model"
	diagcatalog "github.com/keonho-kim/meca/internal/diagnostics/catalog"
	faults "github.com/keonho-kim/meca/internal/diagnostics/fault"
	"github.com/keonho-kim/meca/internal/provider/connection"
	"github.com/keonho-kim/meca/internal/provider/protocol/chat"
)

func nativeControls(kind connection.ProviderKind, r *connection.Reasoning, request modelapi.Request) (Controls, error) {
	controls := Controls{}
	if err := r.CheckEffort(request.ReasoningEffort); err != nil {
		return controls, err
	}
	if request.Thinking == nil {
		if request.ReasoningEffort != "" {
			return controls, faults.Errorf(diagcatalog.ProtocolExplicitThinkingIsRequiredForEffort)
		}
		return controls, nil
	}
	if kind == connection.Claude {
		on := *request.Thinking
		if on && !r.Enabled() || !on && !r.Off {
			return controls, faults.Errorf(diagcatalog.ProtocolThinkingChoiceIsUnavailableForThisClaude)
		}
		mode := "disabled"
		if on {
			mode = "adaptive"
			controls.ReasoningEffort = request.ReasoningEffort
		}
		controls.Think = map[string]string{"type": mode}
		return controls, nil
	}
	if kind == connection.VLLM || kind == connection.SGLang {
		return templateControls(kind, r, request)
	}
	if kind == connection.OpenRouter {
		on := *request.Thinking
		if on && !r.Enabled() || !on && !r.Off {
			return controls, faults.Errorf(diagcatalog.ProtocolThinkingChoiceIsUnavailableForThisOpenRouter)
		}
		controls.Reasoning = &chat.Reasoning{Enabled: on}
		if on {
			controls.Reasoning.Effort = request.ReasoningEffort
		}
		return controls, nil
	}
	if !*request.Thinking {
		if !r.Off {
			return controls, faults.Errorf(diagcatalog.ProtocolThisModelDoesNotAdvertiseThinkingOFF)
		}
		if kind == connection.Ollama {
			controls.Think = false
		} else {
			controls.ReasoningEffort = "none"
		}
		return controls, nil
	}
	if !r.Enabled() {
		return controls, faults.Errorf(diagcatalog.ProtocolThisModelDoesNotAdvertiseThinkingON)
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
		return Controls{}, faults.Errorf(diagcatalog.ProtocolThinkingChoiceIsUnavailableForThisConnection)
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
