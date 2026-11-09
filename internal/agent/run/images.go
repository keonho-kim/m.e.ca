// Purpose: Select image tools and recover from a definitive model image-input rejection.
// Pattern: Request-owned capability projection.
// Usage: Run freezes available tools; the serial owner handles capability failures before another decision.
// Related: internal/agent/run/run.go, internal/agent/context/request.go.
package run

import (
	"context"
	"errors"

	modelapi "github.com/keonho-kim/meca/internal/agent/model"
	runrecord "github.com/keonho-kim/meca/internal/agent/record"
	toolapi "github.com/keonho-kim/meca/internal/agent/tool"
)

func imageDefinitions(definitions []toolapi.Definition, enabled bool) []toolapi.Definition {
	out := make([]toolapi.Definition, 0, len(definitions))
	for _, definition := range definitions {
		if definition.Name != "view_image" || enabled {
			out = append(out, definition)
		}
	}
	return out
}

func (s *execution) recoverImages(ctx context.Context, err error) (bool, error) {
	var unavailable *modelapi.ImageInputError
	if !errors.As(err, &unavailable) || !s.runner.ImageInput {
		return false, nil
	}
	s.runner.ImageInput = false
	setSkillImageInput(s.runner.Tools, false)
	s.definitions = imageDefinitions(s.definitions, false)
	note := "The selected model rejected image input. Images were not available for visual inspection. view_image is unavailable for this connection. Report this limitation when the request requires images."
	if err := s.save(ctx, modelapi.Message{Role: "system", Content: note}); err != nil {
		return true, err
	}
	return true, s.emit(runrecord.Event{Kind: "image_input_unavailable", Text: s.runner.Redact(unavailable.Error())})
}

func setSkillImageInput(executor toolapi.Executor, enabled bool) {
	if tools, ok := executor.(interface{ SetSkillImageInput(bool) }); ok {
		tools.SetSkillImageInput(enabled)
	}
}
