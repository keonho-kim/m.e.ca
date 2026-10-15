// Purpose: Recover limited LiteLLM provider evidence when deployment identities are inaccessible.
// Pattern: HTTP Adapter.
// Usage: LiteLLM identification queries current end-user group metadata with the supplied credential.
// Related: internal/provider/discovery/identify_litellm.go, internal/provider/identity/report.go.
package discovery

import (
	"context"
	"github.com/keonho-kim/v-code/internal/provider/identity"
	"github.com/keonho-kim/v-code/internal/provider/transport"
	"net/http"
	"net/url"
	"slices"
)

func (c *Client) identifyLiteLLMGroup(ctx context.Context, model string) (identity.Report, error) {
	var listing struct {
		Data []struct {
			Group     string   `json:"model_group"`
			Providers []string `json:"providers"`
		} `json:"data"`
	}
	report := identity.Report{Unavailable: true}
	endpoint := c.serverRoot() + "/model_group/info?model_group=" + url.QueryEscape(model)
	if err := transport.ReadMetadata(ctx, c.HTTP, http.MethodGet, endpoint, nil, 16*1024*1024, &listing); err != nil {
		return report, err
	}
	if len(listing.Data) > 10000 {
		return report, nil
	}
	var providers []string
	for _, row := range listing.Data {
		if row.Group != model {
			continue
		}
		for _, provider := range row.Providers {
			if !slices.Contains(providers, provider) {
				providers = append(providers, provider)
			}
			report.Add("/model_group/info.providers", "provider", provider)
		}
	}
	report.Routed = len(providers) > 1
	return report, nil
}
