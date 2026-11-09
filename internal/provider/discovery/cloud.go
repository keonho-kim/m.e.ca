// Purpose: Read authenticated Claude and Gemini model catalogs and advertised serving capabilities.
// Pattern: Bounded first-party metadata adapter.
// Usage: ListModels and PrepareModel use native endpoints without model-name capability guesses.
// Related: internal/provider/discovery/discovery.go, internal/provider/discovery/model_settings.go.
package discovery

import (
	"context"

	"net/url"
	"sort"
	"strings"
	"unicode"
	"unicode/utf8"

	diagcatalog "github.com/keonho-kim/meca/internal/diagnostics/catalog"
	faults "github.com/keonho-kim/meca/internal/diagnostics/fault"
	"github.com/keonho-kim/meca/internal/provider/connection"
)

type supported struct {
	Supported bool `json:"supported"`
}
type claudeModel struct {
	ID           string `json:"id"`
	MaxInput     int    `json:"max_input_tokens"`
	MaxOutput    int    `json:"max_tokens"`
	Capabilities *struct {
		Image    *supported `json:"image_input"`
		Thinking struct {
			Types struct{ Adaptive, Disabled supported } `json:"types"`
		} `json:"thinking"`
		Effort struct{ Low, Medium, High, Xhigh, Max supported } `json:"effort"`
	} `json:"capabilities"`
}

func (row claudeModel) info() connection.ModelInfo {
	info := connection.ModelInfo{ID: row.ID, ContextWindow: row.MaxInput, MaxInput: row.MaxInput, MaxOutput: row.MaxOutput}
	if c := row.Capabilities; c != nil {
		if c.Image != nil {
			info.ImageInput = connection.DeclaredImageInput(c.Image.Supported, "Claude Models API capabilities.image_input")
		}
		r := &connection.Reasoning{On: c.Thinking.Types.Adaptive.Supported, Off: c.Thinking.Types.Disabled.Supported}
		if r.On {
			for _, level := range []struct {
				name   string
				option supported
			}{{"low", c.Effort.Low}, {"medium", c.Effort.Medium}, {"high", c.Effort.High}, {"xhigh", c.Effort.Xhigh}, {"max", c.Effort.Max}} {
				if level.option.Supported {
					r.Levels = append(r.Levels, level.name)
				}
			}
		}
		info.Reasoning = r
	}
	return info
}

type geminiModel struct {
	Thinking *bool    `json:"thinking"`
	Name     string   `json:"name"`
	Input    int      `json:"inputTokenLimit"`
	Output   int      `json:"outputTokenLimit"`
	Methods  []string `json:"supportedGenerationMethods"`
}

func (row geminiModel) info() connection.ModelInfo {
	return connection.ModelInfo{ID: strings.TrimPrefix(row.Name, "models/"), ContextWindow: row.Input, MaxInput: row.Input, MaxOutput: row.Output, ThoughtSummary: row.Thinking}
}

func (c *Client) cloudModels(ctx context.Context) ([]connection.ModelInfo, error) {
	var values []connection.ModelInfo
	cursor := ""
	seen := map[string]bool{}
	for pages := 0; pages < 100; pages++ {
		next := ""
		if c.Connection.Kind == connection.Claude {
			endpoint := c.url("models") + "?limit=1000"
			if cursor != "" {
				endpoint += "&after_id=" + url.QueryEscape(cursor)
			}
			var page struct {
				Data []claudeModel `json:"data"`
				More bool          `json:"has_more"`
				Last string        `json:"last_id"`
			}
			if err := readMetadata(ctx, c, endpoint, &page); err != nil {
				return nil, err
			}
			if page.Data == nil {
				return nil, faults.Errorf(diagcatalog.DiscoveryInvalidClaudeModelCatalog)
			}
			for _, row := range page.Data {
				values = append(values, row.info())
			}
			if page.More {
				next = page.Last
				if next == "" {
					return nil, faults.Errorf(diagcatalog.DiscoveryClaudeModelPageOmittedItsCursor)
				}
			}
		} else {
			endpoint := c.url("models") + "?pageSize=1000"
			if cursor != "" {
				endpoint += "&pageToken=" + url.QueryEscape(cursor)
			}
			var page struct {
				Models []geminiModel `json:"models"`
				Next   string        `json:"nextPageToken"`
			}
			if err := readMetadata(ctx, c, endpoint, &page); err != nil {
				return nil, err
			}
			if page.Models == nil {
				return nil, faults.Errorf(diagcatalog.DiscoveryInvalidGeminiModelCatalog)
			}
			for _, row := range page.Models {
				for _, method := range row.Methods {
					if method == "generateContent" {
						values = append(values, row.info())
						break
					}
				}
			}
			next = page.Next
		}
		if len(values) > 10000 {
			return nil, faults.Errorf(diagcatalog.DiscoveryTooManyProviderModels)
		}
		if next == "" {
			ids := map[string]bool{}
			for _, value := range values {
				if err := checkCloudModel(value); err != nil {
					return nil, err
				}
				if ids[value.ID] {
					return nil, faults.Errorf(diagcatalog.DiscoveryDuplicateProviderModel)
				}
				ids[value.ID] = true
			}
			sort.Slice(values, func(i, j int) bool { return values[i].ID < values[j].ID })
			return values, nil
		}
		if len(next) > 4096 || seen[next] {
			return nil, faults.Errorf(diagcatalog.DiscoveryInvalidProviderModelPageCursor)
		}
		seen[next] = true
		cursor = next
	}
	return nil, faults.Errorf(diagcatalog.DiscoveryProviderModelPaginationExceedsLimit)
}

func (c *Client) cloudModel(ctx context.Context, model string) (connection.ModelInfo, error) {
	var info connection.ModelInfo
	if c.Connection.Kind == connection.Claude {
		var row claudeModel
		if err := readMetadata(ctx, c, c.url("models/")+url.PathEscape(model), &row); err != nil {
			return info, err
		}
		info = row.info()
	} else {
		var row geminiModel
		if err := readMetadata(ctx, c, c.url("models/")+url.PathEscape(strings.TrimPrefix(model, "models/")), &row); err != nil {
			return info, err
		}
		info = row.info()
	}
	if err := checkCloudModel(info); err != nil {
		return info, err
	}
	info.ID = model // Model aliases retain the exact selected request identity.
	return info, nil
}

func checkCloudModel(value connection.ModelInfo) error {
	if !utf8.ValidString(value.ID) || strings.ContainsFunc(value.ID, unicode.IsControl) {
		return faults.Errorf(diagcatalog.DiscoveryInvalidProviderModelID)
	}
	return value.Check()
}
