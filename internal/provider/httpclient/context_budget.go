// Purpose: Apply the configured context budget without modifying discovered server metadata.
// Pattern: Request-local projection.
// Usage: Setup sets the budget before ownership transfer; Generate resolves effective limits.
// Related: internal/provider/httpclient/client.go, internal/provider/connection/context.go.
package httpclient

import "github.com/keonho-kim/v-code/internal/provider/connection"

// SetContextLength is called before generation or transfer to a worker.
func (c *Client) SetContextLength(tokens int) error {
	if err := connection.CheckContextLength(tokens); err != nil {
		return err
	}
	c.contextLength = tokens
	return nil
}

func (c *Client) effectiveModel() *connection.ModelInfo {
	info := connection.ModelInfo{}
	if c.modelInfo != nil {
		info = *c.modelInfo
	}
	info.ContextWindow = connection.EffectiveContextLength(c.contextLength, info.ContextWindow)
	return &info
}
