// Purpose: Read OpenRouter model limits and advertised reasoning controls without model-name rules.
// Pattern: HTTP metadata adapter.
// Usage: ListModels supplies model selection and PrepareModel with OpenRouter metadata.
// Related: internal/provider/discovery/discovery.go, internal/provider/connection/reasoning.go.
package discovery

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/keonho-kim/meca/internal/provider/connection"
)

func (c *Client) openRouterModels(ctx context.Context) ([]connection.ModelInfo, error) {
	var listing struct {
		Data []struct {
			ID           string `json:"id"`
			Architecture struct {
				Input []string `json:"input_modalities"`
			} `json:"architecture"`
			ContextLength int `json:"context_length"`
			TopProvider   struct {
				ContextLength int `json:"context_length"`
				MaxCompletion int `json:"max_completion_tokens"`
			} `json:"top_provider"`
			Reasoning *struct {
				Efforts   json.RawMessage `json:"supported_efforts"`
				Default   string          `json:"default_effort"`
				Mandatory bool            `json:"mandatory"`
				Enabled   *bool           `json:"default_enabled"`
			} `json:"reasoning"`
		} `json:"data"`
	}
	if err := readMetadata(ctx, c, c.url("models"), &listing); err != nil {
		return nil, err
	}
	if listing.Data == nil || len(listing.Data) > 10000 {
		return nil, fmt.Errorf("invalid OpenRouter model catalog")
	}
	models := make([]connection.ModelInfo, 0, len(listing.Data))
	seen := make(map[string]bool)
	for _, row := range listing.Data {
		info := connection.ModelInfo{ID: row.ID, ContextWindow: row.ContextLength, MaxOutput: row.TopProvider.MaxCompletion}
		if row.Architecture.Input != nil {
			vision := false
			for _, mode := range row.Architecture.Input {
				vision = vision || mode == "image"
			}
			info.ImageInput = connection.DeclaredImageInput(vision, "/api/v1/models.architecture.input_modalities")
		}
		if strings.ContainsAny(row.ID, "\x00\r\n\x1b") || seen[row.ID] || row.TopProvider.ContextLength < 0 {
			return nil, fmt.Errorf("invalid or duplicate OpenRouter model ID or limits")
		}
		if row.TopProvider.ContextLength > 0 && (info.ContextWindow == 0 || row.TopProvider.ContextLength < info.ContextWindow) {
			info.ContextWindow = row.TopProvider.ContextLength
		}
		if r := row.Reasoning; r != nil {
			options := &connection.Reasoning{On: true, Off: !r.Mandatory, DefaultOn: r.Enabled}
			if string(r.Efforts) == "null" {
				options.Levels = []string{"max", "xhigh", "high", "medium", "low", "minimal"}
			} else if len(r.Efforts) > 0 {
				var levels []string
				if err := json.Unmarshal(r.Efforts, &levels); err != nil {
					return nil, fmt.Errorf("OpenRouter reasoning efforts: %w", err)
				}
				for _, level := range levels {
					if level != "none" {
						options.Levels = append(options.Levels, level)
					}
				}
			}
			if r.Default != "none" {
				options.Default = r.Default
			}
			info.Reasoning = options
		}
		if err := info.Check(); err != nil {
			return nil, fmt.Errorf("OpenRouter model %q: %w", row.ID, err)
		}
		seen[row.ID] = true
		models = append(models, info)
	}
	sort.Slice(models, func(i, j int) bool { return models[i].ID < models[j].ID })
	return models, nil
}
