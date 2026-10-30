// Purpose: Dispatch provider metadata discovery and validate the resulting model binding.
// Pattern: Simple Module.
// Usage: Setup calls PrepareModel before model diagnostics.
// Related: internal/provider/discovery/ollama_settings.go, internal/provider/discovery/lmstudio_settings.go.
package discovery

import (
	"context"
	"github.com/keonho-kim/meca/internal/provider/connection"
)

func (c *Client) PrepareModel(ctx context.Context, model string) (connection.ModelInfo, error) {
	info := connection.ModelInfo{ID: model}
	if c.imageModel == model {
		info.ImageInput = c.imageInput
	}
	switch c.Connection.Kind {
	case connection.Ollama:
		var err error
		info, err = c.prepareOllama(ctx, model)
		if err != nil {
			return info, err
		}
	case connection.LMStudio:
		var err error
		info, err = c.prepareLmstudio(ctx, model)
		if err != nil {
			return info, err
		}
	case connection.LiteLLM:
		var err error
		info, err = c.liteLLMLimits(ctx, model)
		if err != nil {
			return info, err
		}
	default:
		models, err := c.ListModels(ctx)
		if err != nil {
			return info, err
		}
		for _, row := range models {
			if row.ID == model {
				info = row
				break
			}
		}
	}
	// SGLang's generic model list carries limits; its selected /model_info
	// response owns image capability evidence.
	if c.imageModel == model {
		info.ImageInput = c.imageInput
	}
	if err := info.Check(); err != nil {
		return info, err
	}
	return info, nil
}
