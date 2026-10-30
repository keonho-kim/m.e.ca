// Purpose: Read ollama model context and native reasoning metadata.
// Pattern: Metadata adapter.
// Usage: PrepareModel dispatches this provider metadata query.
// Related: internal/provider/discovery/model_settings.go.
package discovery

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/keonho-kim/meca/internal/provider/connection"
)

func (c *Client) prepareOllama(ctx context.Context, model string) (connection.ModelInfo, error) {
	info := connection.ModelInfo{ID: model}

	row, err := c.readOllama(ctx, model)
	if err != nil {
		return info, err
	}

	if row.Capabilities != nil {
		vision := false
		for _, cap := range row.Capabilities {
			vision = vision || cap == "vision"
		}
		info.ImageInput = connection.DeclaredImageInput(vision, "/api/show.capabilities")
	}
	if row.Thinking.Values != nil {
		control := &connection.Reasoning{}
		for _, value := range row.Thinking.Values {
			switch string(value) {
			case "true":
				control.On = true
			case "false":
				control.Off = true
			default:
				var level string
				if err := json.Unmarshal(value, &level); err != nil {
					return info, fmt.Errorf("invalid native reasoning value: %w", err)
				}
				control.Levels = append(control.Levels, level)
			}
		}
		if len(row.Thinking.Default) > 0 && row.Thinking.Default[0] == '"' {
			if err := json.Unmarshal(row.Thinking.Default, &control.Default); err != nil {
				return info, err
			}
		} else if string(row.Thinking.Default) == "true" || string(row.Thinking.Default) == "false" {
			on := string(row.Thinking.Default) == "true"
			control.DefaultOn = &on
		}
		info.Reasoning = control
	}
	contextWindow, err := c.ollamaContext(ctx, model)
	if err != nil {
		if ctx.Err() != nil {
			return info, ctx.Err()
		}
		info.Notice = "Running context unavailable: " + c.HTTP.FailureReason(err)
	} else {
		info.ContextWindow = contextWindow
	}

	return info, nil
}
