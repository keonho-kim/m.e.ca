// Purpose: Extract original roots from the selected OpenAI-compatible model card.
// Pattern: HTTP Adapter.
// Usage: vLLM and OpenRouter use Client.identifyModels.
// Related: internal/provider/discovery/identify.go, internal/provider/identity/report.go.
package discovery

import (
	"context"
	"github.com/keonho-kim/meca/internal/provider/connection"
	"github.com/keonho-kim/meca/internal/provider/identity"
)

func (c *Client) identifyModels(ctx context.Context, model string) (identity.Report, error) {
	var listing struct {
		Data []struct {
			ID, Root, Parent string
			Owner            string `json:"owned_by"`
			Canonical        string `json:"canonical_slug"`
			Name             string `json:"name"`
		} `json:"data"`
	}
	var report identity.Report
	if err := readMetadata(ctx, c, c.url("models"), &listing); err != nil {
		return report, err
	}
	if listing.Data == nil || len(listing.Data) > 10000 {
		report.Unavailable = true
		return report, nil
	}
	matched := false
	for _, row := range listing.Data {
		if row.ID != model {
			continue
		}
		matched = true
		if c.Connection.Kind == connection.OpenRouter {
			report.Add("/api/v1/models.canonical_slug", "original", row.Canonical)
			report.Add("/api/v1/models.name", "label", row.Name)
			continue
		}
		if row.Root != row.ID {
			report.Add("/v1/models.root", "original", row.Root)
		}
		report.Add("/v1/models.parent", "parent", row.Parent)
		if row.Parent != "" {
			report.Incomplete = true
		} // An adapter's parent is not the adapted checkpoint.
		report.Add("/v1/models.owned_by", "publisher", row.Owner)
	}
	report.Unavailable = !matched
	return report, nil
}
