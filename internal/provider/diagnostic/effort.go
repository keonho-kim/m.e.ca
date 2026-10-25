// Purpose: Probe selected Thinking effort for complete answers and separate reasoning.
// Pattern: Diagnostic use case.
// Usage: Explicit TUI and headless connection diagnostics invoke Effort through the generator contract.
// Related: internal/provider/diagnostic/check.go, internal/agent/model/model.go.
package diagnostic

import (
	"context"
	"fmt"
	"github.com/keonho-kim/meca/internal/agent/inference"
	modelapi "github.com/keonho-kim/meca/internal/agent/model"
)

func probeThinking(ctx context.Context, c modelapi.Generator, model, effort string, on bool, settings *inference.Settings) (bool, error) {
	parameters, err := inference.Resolve(settings, inference.Diagnostic)
	if err != nil {
		return false, err
	}
	reasoning, answer, done := false, false, false
	err = c.Generate(ctx, parameters.Apply(modelapi.Request{ModelID: model, Thinking: &on, ReasoningEffort: effort, Messages: []modelapi.Message{{Role: "user", Content: "Think briefly through 17 times 23, then answer with only the result."}}}), func(event modelapi.Event) error {
		if event.Retry != nil {
			reasoning, answer, done = false, false, false
			return ctx.Err()
		}
		reasoning = reasoning || event.Thinking != ""
		answer = answer || event.Delta != ""
		done = done || event.Done
		return ctx.Err()
	})
	if err == nil && (!answer || !done) {
		err = fmt.Errorf("thinking diagnostic did not produce a complete answer")
	}
	return reasoning, err
}

func Effort(ctx context.Context, c modelapi.Generator, model, effort string, settings *inference.Settings) error {
	if effort == "" {
		return ctx.Err()
	}
	supported, err := probeThinking(ctx, c, model, effort, true, settings)
	if err != nil {
		return err
	}
	if !supported {
		return fmt.Errorf("thinking effort diagnostic did not produce a complete answer with separate reasoning")
	}
	return nil
}
