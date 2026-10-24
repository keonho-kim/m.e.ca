// Purpose: Read active Ollama context and conservative LiteLLM deployment limits.
// Pattern: Provider metadata adapters.
// Usage: PrepareModel binds these serving limits before client ownership transfers.
// Related: internal/provider/discovery/model_settings.go, internal/provider/connection/models.go.
package discovery

import (
	"context"
	"fmt"
	"github.com/keonho-kim/meca/internal/provider/connection"
	"github.com/keonho-kim/meca/internal/provider/transport"
	"net/http"
)

func (c *Client) ollamaContext(ctx context.Context, model string) (int, error) {
	var listing struct {
		Models []struct {
			Name, Model string
			Context     int `json:"context_length"`
		} `json:"models"`
	}
	if err := transport.ReadMetadata(ctx, c.HTTP, http.MethodGet, c.url("ps"), nil, 16*1024*1024, &listing); err != nil {
		return 0, err
	}
	if listing.Models == nil || len(listing.Models) > 10000 {
		return 0, fmt.Errorf("invalid running model list")
	}
	size := 0
	for _, row := range listing.Models {
		if row.Name != model && row.Model != model {
			continue
		}
		if row.Context < 0 || row.Context > 1<<40 {
			return 0, fmt.Errorf("invalid running model context")
		}
		if size != 0 && size != row.Context {
			return 0, fmt.Errorf("ambiguous running model context")
		}
		size = row.Context
	}
	return size, nil
}

func (c *Client) liteLLMLimits(ctx context.Context, model string) (connection.ModelInfo, error) {
	info := connection.ModelInfo{ID: model}
	var listing struct {
		Data []struct {
			Name string `json:"model_name"`
			Info struct {
				Input  int `json:"max_input_tokens"`
				Output int `json:"max_output_tokens"`
			} `json:"model_info"`
		} `json:"data"`
	}
	if err := transport.ReadMetadata(ctx, c.HTTP, http.MethodGet, c.serverRoot()+"/model/info", nil, 16*1024*1024, &listing); err != nil {
		return info, err
	}
	if listing.Data == nil || len(listing.Data) > 10000 {
		return info, fmt.Errorf("invalid deployment model list")
	}
	matches := 0
	unknownInput, unknownOutput := false, false
	for _, row := range listing.Data {
		if row.Name != model {
			continue
		}
		matches++
		if row.Info.Input < 0 || row.Info.Output < 0 || row.Info.Input > 1<<40 || row.Info.Output > 1<<40 {
			return info, fmt.Errorf("invalid deployment token limits")
		}
		unknownInput = unknownInput || row.Info.Input == 0
		unknownOutput = unknownOutput || row.Info.Output == 0
		if info.MaxInput == 0 || row.Info.Input < info.MaxInput {
			info.MaxInput = row.Info.Input
		}
		if info.MaxOutput == 0 || row.Info.Output < info.MaxOutput {
			info.MaxOutput = row.Info.Output
		}
	}
	if matches == 0 {
		return info, fmt.Errorf("model limits not advertised for selected alias")
	}
	if unknownInput {
		info.MaxInput = 0
	}
	if unknownOutput {
		info.MaxOutput = 0
	}
	return info, nil
}
