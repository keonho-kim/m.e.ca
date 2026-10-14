// Purpose: Extract Ollama family, quantization and declared capabilities without retaining prompts.
// Pattern: HTTP Adapter.
// Usage: Client.Identify posts the selected alias to /api/show.
// Related: internal/provider/discovery/identify.go, internal/provider/diagnostic/control.go.
package discovery

import (
	"bytes"
	"context"
	"encoding/json"
	"github.com/keonho-kim/v-code/internal/provider/identity"
	"github.com/keonho-kim/v-code/internal/provider/transport"
	"net/http"
)

type ollamaModel struct {
	Details struct {
		Family       string   `json:"family"`
		Families     []string `json:"families"`
		Parent       string   `json:"parent_model"`
		Parameters   string   `json:"parameter_size"`
		Quantization string   `json:"quantization_level"`
	} `json:"details"`
	ModelInfo   map[string]json.RawMessage `json:"model_info"`
	RemoteModel string                     `json:"remote_model"`
	Thinking    struct {
		Values  []json.RawMessage `json:"values"`
		Default json.RawMessage   `json:"default"`
	} `json:"thinking"`
	Manifests []struct {
		Digest   string
		Selected bool
	} `json:"manifests"`
}

func (c *Client) identifyOllama(ctx context.Context, model string) (identity.Report, error) {
	var info ollamaModel
	var report identity.Report
	body, err := json.Marshal(map[string]string{"model": model})
	if err != nil {
		return report, err
	}
	if err := transport.ReadMetadata(ctx, c.HTTP, http.MethodPost, c.url("show"), bytes.NewReader(body), 4*1024*1024, &info); err != nil {
		return report, err
	}
	report.Add("/api/show.details.family", "family", info.Details.Family)
	for _, family := range info.Details.Families {
		report.Add("/api/show.details.families", "family", family)
	}
	report.Add("/api/show.details.parent_model", "parent", info.Details.Parent)
	report.Add("/api/show.details.parameter_size", "parameters", info.Details.Parameters)
	report.Add("/api/show.details.quantization_level", "quantization", info.Details.Quantization)
	report.Add("/api/show.remote_model", "original", info.RemoteModel)
	for _, field := range []string{"general.architecture", "general.basename", "general.source.repo_url", "general.base_model.0.repo_url"} {
		var value string
		if json.Unmarshal(info.ModelInfo[field], &value) == nil {
			kind := "architecture"
			switch field {
			case "general.basename":
				kind = "label"
			case "general.source.repo_url":
				kind = "original"
			case "general.base_model.0.repo_url":
				kind = "parent"
			}
			report.Add("/api/show.model_info."+field, kind, value)
		}
	}
	for _, manifest := range info.Manifests {
		if manifest.Selected {
			report.Add("/api/show.manifests.digest", "digest", manifest.Digest)
		}
	}
	thinkingFacts(&report, "/api/show.thinking.values", info.Thinking.Values)
	return report, nil
}
