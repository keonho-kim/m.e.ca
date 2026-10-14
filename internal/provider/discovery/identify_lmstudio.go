// Purpose: Resolve LM Studio instance aliases to native model keys and advertised architecture.
// Pattern: HTTP Adapter.
// Usage: Client.Identify reads /api/v1/models regardless of the chat API root.
// Related: internal/provider/discovery/identify.go, internal/provider/connection/routes.go.
package discovery

import (
	"context"
	"encoding/json"
	"github.com/keonho-kim/v-code/internal/provider/identity"
	"github.com/keonho-kim/v-code/internal/provider/transport"
	"net/http"
)

type lmModelListing struct {
	Models []struct {
		Key, Type, Publisher, Architecture string
		Display                            string                `json:"display_name"`
		Parameters                         string                `json:"params_string"`
		Quantization                       struct{ Name string } `json:"quantization"`
		ContextWindow                      int                   `json:"max_context_length"`
		Instances                          []struct {
			ID     string
			Config struct {
				ContextLength int `json:"context_length"`
			}
		} `json:"loaded_instances"`
		Capabilities struct {
			Reasoning struct {
				Options []json.RawMessage `json:"allowed_options"`
				Default string            `json:"default"`
			} `json:"reasoning"`
		} `json:"capabilities"`
	} `json:"models"`
}

func (c *Client) identifyLMStudio(ctx context.Context, model string) (identity.Report, error) {
	var listing lmModelListing
	var report identity.Report
	if err := transport.ReadMetadata(ctx, c.HTTP, http.MethodGet, c.serverRoot()+"/api/v1/models", nil, 4*1024*1024, &listing); err != nil {
		return report, err
	}
	if listing.Models == nil || len(listing.Models) > 10000 {
		report.Unavailable = true
		return report, nil
	}
	matched := false
	for _, row := range listing.Models {
		if row.Type != "llm" {
			continue
		}
		matches := row.Key == model
		for _, instance := range row.Instances {
			matches = matches || instance.ID == model
		}
		if !matches {
			continue
		}
		matched = true
		report.Add("/api/v1/models.key", "original", row.Key)
		report.Add("/api/v1/models.publisher", "publisher", row.Publisher)
		report.Add("/api/v1/models.display_name", "label", row.Display)
		report.Add("/api/v1/models.architecture", "architecture", row.Architecture)
		report.Add("/api/v1/models.params_string", "parameters", row.Parameters)
		report.Add("/api/v1/models.quantization.name", "quantization", row.Quantization.Name)
		thinkingFacts(&report, "/api/v1/models.capabilities.reasoning", row.Capabilities.Reasoning.Options)
	}
	report.Unavailable = !matched
	return report, nil
}
