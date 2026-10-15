// Purpose: Discover bounded server model catalogs through documented provider endpoints.
// Pattern: HTTP discovery adapter.
// Usage: Client.ListModels is used by provider setup.
// Related: internal/provider/discovery/client.go, internal/provider/connection/models.go.

package discovery

import (
	"context"
	"fmt"
	modelconn "github.com/keonho-kim/v-code/internal/provider/connection"
	"github.com/keonho-kim/v-code/internal/provider/transport"
	"net/http"
	"sort"
	"strings"
)

func (client *Client) ListModels(ctx context.Context) (models []modelconn.ModelInfo, err error) {
	endpoint := client.url("models")
	if client.Connection.Kind == modelconn.Ollama {
		endpoint = client.url("tags")
	}
	if client.Connection.Kind == modelconn.LMStudio {
		endpoint = client.serverRoot() + "/api/v1/models"
	}
	var listing struct {
		Models []struct {
			Name string `json:"name"`
			Key  string `json:"key"`
			Type string `json:"type"`
		} `json:"models"`
		Data []struct {
			ID            string `json:"id"`
			ContextWindow int    `json:"max_model_len"`
		} `json:"data"`
	}
	if err := transport.ReadMetadata(ctx, client.HTTP, http.MethodGet, endpoint, nil, 16*1024*1024, &listing); err != nil {
		return nil, err
	}
	if listing.Models == nil && listing.Data == nil {
		return nil, fmt.Errorf("unsupported model list format; enter Model ID manually")
	}
	seen := make(map[string]bool)
	appendModel := func(id string, contextWindow int) error {
		if id == "" || len(id) > 512 || strings.ContainsAny(id, "\x00\r\n\x1b") {
			return fmt.Errorf("invalid Model ID")
		}
		if !seen[id] {
			models = append(models, modelconn.ModelInfo{ID: id, ContextWindow: contextWindow})
			seen[id] = true
		}
		if len(models) > 10000 {
			return fmt.Errorf("too many models")
		}
		return nil
	}
	for _, model := range listing.Models {
		if model.Type != "" && model.Type != "llm" {
			continue
		}
		id := model.Name
		if model.Key != "" {
			id = model.Key
		}
		if err := appendModel(id, 0); err != nil {
			return nil, err
		}
	}
	for _, model := range listing.Data {
		if err := appendModel(model.ID, model.ContextWindow); err != nil {
			return nil, err
		}
	}
	sort.Slice(models, func(i, j int) bool { return models[i].ID < models[j].ID })
	return models, nil
}
