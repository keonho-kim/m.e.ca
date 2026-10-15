// Purpose: Read lmstudio model context and native reasoning metadata.
// Pattern: Metadata adapter.
// Usage: PrepareModel dispatches this provider metadata query.
// Related: internal/provider/discovery/model_settings.go.
package discovery

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/keonho-kim/v-code/internal/provider/connection"
	"github.com/keonho-kim/v-code/internal/provider/transport"
	"net/http"
)

func (c *Client) prepareLmstudio(ctx context.Context, model string) (connection.ModelInfo, error) {
	info := connection.ModelInfo{ID: model}

	var listing lmModelListing
	if err := transport.ReadMetadata(ctx, c.HTTP, http.MethodGet, c.serverRoot()+"/api/v1/models", nil, 16*1024*1024, &listing); err != nil {
		return info, err
	}
	if listing.Models == nil || len(listing.Models) > 10000 {
		return info, fmt.Errorf("invalid native model list")
	}
	matches := 0
	for _, row := range listing.Models {
		if row.Type != "llm" {
			continue
		}
		match := row.Key == model
		contextWindow := 0
		for _, instance := range row.Instances {
			if instance.ID == model {
				match = true
				contextWindow = instance.Config.ContextLength
				break
			}
			if row.Key == model && instance.Config.ContextLength > 0 && (contextWindow == 0 || instance.Config.ContextLength < contextWindow) {
				contextWindow = instance.Config.ContextLength
			}
		}
		if !match {
			continue
		}
		matches++
		// A model's theoretical maximum is not the loaded instance's usable context.
		info.ContextWindow = contextWindow
		if row.Capabilities.Reasoning.Options != nil {
			control := &connection.Reasoning{}
			for _, raw := range row.Capabilities.Reasoning.Options {
				var value string
				if err := json.Unmarshal(raw, &value); err != nil {
					return info, err
				}
				switch value {
				case "off":
					control.Off = true
				case "on":
					control.On = true
				case "low", "medium", "high":
					control.Levels = append(control.Levels, value)
				default:
					return info, fmt.Errorf("unsupported LM Studio reasoning option %q", value)
				}
			}
			if value := row.Capabilities.Reasoning.Default; value != "on" && value != "off" {
				control.Default = value
			}
			info.Reasoning = control
		}
	}
	if matches != 1 {
		return info, fmt.Errorf("LM Studio model must identify exactly one model entry")
	}

	return info, nil
}
