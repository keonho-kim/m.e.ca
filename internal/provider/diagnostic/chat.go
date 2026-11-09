// Purpose: Verify an ordinary model response completes without imposing a tool or answer format.
// Pattern: Connection diagnostic use case.
// Usage: Check runs this probe before paired Thinking diagnostics.
// Related: internal/provider/diagnostic/check.go, internal/agent/model/model.go.
package diagnostic

import (
	"context"

	"strings"

	"github.com/keonho-kim/meca/internal/agent/inference"
	modelapi "github.com/keonho-kim/meca/internal/agent/model"
	diagcatalog "github.com/keonho-kim/meca/internal/diagnostics/catalog"
	faults "github.com/keonho-kim/meca/internal/diagnostics/fault"
	modelconn "github.com/keonho-kim/meca/internal/provider/connection"
)

func checkChat(ctx context.Context, client modelapi.Generator, parameters inference.Parameters, model string, thinking *bool) (modelconn.Capabilities, error) {
	caps := modelconn.Capabilities{}
	answer, done, reasoning, received := false, false, false, 0
	request := parameters.Apply(modelapi.Request{ModelID: model, Thinking: thinking, Messages: []modelapi.Message{{Role: "user", Content: "Please reply with a brief greeting."}}})
	err := client.Generate(ctx, request, func(event modelapi.Event) error {
		if event.Retry != nil {
			answer, done, reasoning, received = false, false, false, 0
			caps = modelconn.Capabilities{}
			return ctx.Err()
		}
		received += len(event.Delta) + len(event.Thinking)
		if received > modelapi.MaxResponseBytes {
			return faults.Errorf(diagcatalog.DiagnosticConnectionDiagnosticResponseExceedsTheResponseLimit)
		}
		if len(event.Calls) > 0 {
			return faults.Errorf(diagcatalog.DiagnosticConnectionDiagnosticReceivedANativeToolCall)
		}
		answer = answer || strings.TrimSpace(event.Delta) != ""
		reasoning = reasoning || event.Thinking != ""
		if event.Done {
			done, caps.Streaming = true, event.Streaming
		}
		return ctx.Err()
	})
	if err != nil {
		return caps, faults.Errorf(diagcatalog.DiagnosticConnectionDiagnosticFailed, err)
	}
	if !answer || !done {
		return caps, faults.Errorf(diagcatalog.DiagnosticConnectionDiagnosticDidNotProduceAComplete)
	}
	if thinking != nil && !*thinking && reasoning {
		return caps, faults.Errorf(diagcatalog.DiagnosticServerEmittedReasoningWithThinkingOFFVerify)
	}
	caps.Chat = true
	return caps, nil
}
