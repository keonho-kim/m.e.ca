// Purpose: Bind metadata discovery to HTTP integration fixtures.
// Pattern: Test fixture.
// Usage: Provider client tests use the same test server for setup and generation.
// Related: internal/provider/discovery/client.go.
package httpclient

import (
	"context"
	"github.com/keonho-kim/v-code/internal/provider/connection"
	"github.com/keonho-kim/v-code/internal/provider/diagnostic"
	"github.com/keonho-kim/v-code/internal/provider/discovery"
)

func catalogFor(c *Client) *discovery.Client {
	return &discovery.Client{Connection: c.connection, HTTP: c.transport}
}
func prepareModel(c *Client, ctx context.Context, model string) (connection.ModelInfo, error) {
	info, err := catalogFor(c).PrepareModel(ctx, model)
	if err == nil {
		err = c.BindModel(info)
	}
	return info, err
}

func thinkingControl(c *Client, ctx context.Context, model string) (bool, error) {
	if err := ctx.Err(); err != nil {
		return false, err
	}
	if (c.connection.Kind == connection.Ollama || c.connection.Kind == connection.LMStudio) && c.modelInfo == nil {
		if _, err := prepareModel(c, ctx, model); err != nil {
			return false, err
		}
	}
	return diagnostic.ThinkingControl(c.connection.Kind, c.modelInfo), nil
}
func diagnose(c *Client, ctx context.Context, model string) (connection.Capabilities, error) {
	supported, err := thinkingControl(c, ctx, model)
	if err != nil {
		return connection.Capabilities{}, err
	}
	nativeOn := c.modelInfo != nil && c.modelInfo.Reasoning != nil && c.modelInfo.Reasoning.Enabled()
	return diagnostic.Check(ctx, c, model, supported, nativeOn, nil)
}
