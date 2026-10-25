// Purpose: Extract SGLang weight identity and architecture while checking the serving alias.
// Pattern: HTTP Adapter.
// Usage: Client.Identify queries the current /model_info endpoint.
// Related: internal/provider/discovery/identify.go, internal/provider/identity/report.go.
package discovery

import (
	"context"
	"github.com/keonho-kim/meca/internal/provider/identity"
)

func (c *Client) identifySGLang(ctx context.Context, model string) (identity.Report, error) {
	var info struct {
		Path          string   `json:"model_path"`
		Served        string   `json:"served_model_name"`
		Tokenizer     string   `json:"tokenizer_path"`
		Type          string   `json:"model_type"`
		Architectures []string `json:"architectures"`
	}
	var report identity.Report
	if err := readMetadata(ctx, c, c.serverRoot()+"/model_info", &info); err != nil {
		return report, err
	}
	if info.Served != model && (info.Served != "" || info.Path != model) {
		report.Unavailable = true
		report.Incomplete = true
		return report, nil
	}
	report.Add("/model_info.model_path", "original", info.Path)
	report.Add("/model_info.tokenizer_path", "tokenizer", info.Tokenizer)
	report.Add("/model_info.model_type", "family", info.Type)
	for _, architecture := range info.Architectures {
		report.Add("/model_info.architectures", "architecture", architecture)
	}
	return report, nil
}
