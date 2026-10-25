// Purpose: Describe all LiteLLM upstream identities for an alias without selecting one deployment.
// Pattern: HTTP Adapter.
// Usage: Client.Identify reads the documented proxy model-info endpoint.
// Related: internal/provider/discovery/identify.go, internal/provider/identity/report.go.
package discovery

import (
	"context"
	"github.com/keonho-kim/meca/internal/provider/identity"
	"slices"
	"strings"
)

func (c *Client) identifyLiteLLM(ctx context.Context, model string) (identity.Report, error) {
	var listing struct {
		Data []struct {
			Name   string                 `json:"model_name"`
			Params struct{ Model string } `json:"litellm_params"`
		} `json:"data"`
	}
	var report identity.Report
	if err := readMetadata(ctx, c, c.serverRoot()+"/model/info", &listing); err != nil {
		return c.identifyLiteLLMGroup(ctx, model)
	}
	if listing.Data == nil || len(listing.Data) > 10000 {
		report.Unavailable = true
		return report, nil
	}
	var upstreams []string
	matched := false
	for _, row := range listing.Data {
		if row.Name != model {
			continue
		}
		matched = true
		upstream := row.Params.Model
		if upstream == "" {
			report.Incomplete = true
			continue
		}
		if !slices.Contains(upstreams, upstream) {
			upstreams = append(upstreams, upstream)
		}
		report.Add("/model/info.litellm_params.model", "upstream", upstream)
		provider, original, found := strings.Cut(upstream, "/")
		if !found {
			original = upstream
			provider = ""
		}
		// These prefixes declare a transport. Unrecognized prefixes remain part of the model ID.
		switch provider {
		case "huggingface", "openai", "hosted_vllm", "vllm", "sglang", "ollama", "ollama_chat", "lm_studio", "deepseek", "gemini":
			report.Add("/model/info.litellm_params.model", "provider", provider)
		case "":
		default:
			original = upstream
		}
		if provider == "auto_router" {
			report.Routed = true
			report.Incomplete = true
		}
		report.Add("/model/info.litellm_params.model", "original", original)
	}
	report.Routed = report.Routed || len(upstreams) > 1
	report.Unavailable = !matched
	if !matched {
		return c.identifyLiteLLMGroup(ctx, model)
	}
	return report, nil
}
