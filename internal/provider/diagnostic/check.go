// Purpose: Verify ordinary response completion and paired Thinking behavior through a model generator.
// Pattern: Diagnostic use case.
// Usage: Explicit TUI and headless diagnostics call Check without executing local tools.
// Related: internal/provider/diagnostic/chat.go, internal/provider/diagnostic/control.go, internal/provider/diagnostic/effort.go.
package diagnostic

import (
	"context"

	"github.com/keonho-kim/meca/internal/agent/inference"
	modelapi "github.com/keonho-kim/meca/internal/agent/model"
	diagcatalog "github.com/keonho-kim/meca/internal/diagnostics/catalog"
	faults "github.com/keonho-kim/meca/internal/diagnostics/fault"
	modelconn "github.com/keonho-kim/meca/internal/provider/connection"
)

func Check(ctx context.Context, client modelapi.Generator, model string, supported, nativeOn bool, settings *inference.Settings) (modelconn.Capabilities, error) {
	caps := modelconn.Capabilities{}
	var thinking *bool
	parameters, err := inference.Resolve(settings, inference.Diagnostic)
	if err != nil {
		return caps, err
	}
	if supported {
		off := false
		thinking = &off
	}
	caps, err = checkChat(ctx, client, parameters, model, thinking)
	if err != nil {
		return caps, err
	}
	if supported || nativeOn {
		caps.Reasoning, err = probeThinking(ctx, client, model, "", true, settings)
		if err != nil {
			return caps, faults.Errorf(diagcatalog.DiagnosticThinkingONDiagnosticFailed, err)
		}
	}
	if supported && !caps.Reasoning {
		return caps, faults.Errorf(diagcatalog.DiagnosticThinkingONDiagnosticDidNotProduceSeparate)
	}
	if supported && caps.Reasoning {
		reasoning, err := probeThinking(ctx, client, model, "", false, settings)
		if err != nil {
			return caps, faults.Errorf(diagcatalog.DiagnosticThinkingOFFDiagnosticFailed, err)
		}
		if reasoning {
			return caps, faults.Errorf(diagcatalog.DiagnosticServerEmittedReasoningWithThinkingOFFOn)
		}
	}
	caps.Thinking = supported && caps.Reasoning
	return caps, nil
}
