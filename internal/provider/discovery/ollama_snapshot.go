// Purpose: Share one selected Ollama model response across identity and capability projections.
// Pattern: Preparation-owned typed snapshot.
// Usage: identifyOllama and prepareOllama borrow the same immutable metadata value.
// Related: internal/provider/discovery/identify_ollama.go, internal/provider/discovery/ollama_settings.go.
package discovery

import (
	"bytes"
	"context"
	"encoding/json"
	"github.com/keonho-kim/meca/internal/provider/transport"
	"net/http"
)

func (c *Client) readOllama(ctx context.Context, model string) (ollamaModel, error) {
	if err := ctx.Err(); err != nil {
		return ollamaModel{}, err
	}
	if c.ollamaName == model && c.ollamaSnapshot != nil {
		return *c.ollamaSnapshot, nil
	}
	body, err := json.Marshal(map[string]string{"model": model})
	if err != nil {
		return ollamaModel{}, err
	}
	var value ollamaModel
	if err := transport.ReadMetadata(ctx, c.HTTP, http.MethodPost, c.url("show"), bytes.NewReader(body), 16*1024*1024, &value); err != nil {
		return value, err
	}
	c.ollamaName, c.ollamaSnapshot = model, &value
	return value, nil
}
