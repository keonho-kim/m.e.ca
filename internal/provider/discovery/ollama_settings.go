// Purpose: Read ollama model context and native reasoning metadata.
// Pattern: Metadata adapter.
// Usage: PrepareModel dispatches this provider metadata query.
// Related: internal/provider/discovery/model_settings.go.
package discovery

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"github.com/keonho-kim/v-code/internal/provider/connection"
	"github.com/keonho-kim/v-code/internal/provider/transport"
	"net/http"
)

func (c *Client) prepareOllama(ctx context.Context, model string) (connection.ModelInfo, error) {
	info := connection.ModelInfo{ID: model}

	payload, err := json.Marshal(map[string]string{"model": model})
	if err != nil {
		return info, err
	}
	var row ollamaModel
	if err := transport.ReadMetadata(ctx, c.HTTP, http.MethodPost, c.url("show"), bytes.NewReader(payload), 4*1024*1024, &row); err != nil {
		return info, err
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
