// Purpose: Verify an ordinary model response completes without imposing a tool or answer format.
// Pattern: Connection diagnostic use case.
// Usage: Check runs this probe before paired Thinking diagnostics.
// Related: internal/provider/diagnostic/check.go, internal/agent/model/model.go.
package diagnostic

import (
	"context"
	"fmt"
	"strings"

	"github.com/keonho-kim/meca/internal/agent/inference"
	modelapi "github.com/keonho-kim/meca/internal/agent/model"
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
			return fmt.Errorf("connection diagnostic response exceeds the response limit")
		}
		if len(event.Calls) > 0 {
			return fmt.Errorf("connection diagnostic received a native tool call instead of an answer")
		}
		answer = answer || strings.TrimSpace(event.Delta) != ""
		reasoning = reasoning || event.Thinking != ""
		if event.Done {
			done, caps.Streaming = true, event.Streaming
		}
		return ctx.Err()
	})
	if err != nil {
		return caps, fmt.Errorf("connection diagnostic failed: %w", err)
	}
	if !answer || !done {
		return caps, fmt.Errorf("connection diagnostic did not produce a complete nonempty answer")
	}
	if thinking != nil && !*thinking && reasoning {
		return caps, fmt.Errorf("server emitted reasoning with Thinking OFF; verify its chat template and reasoning parser")
	}
	caps.Chat = true
	return caps, nil
}
