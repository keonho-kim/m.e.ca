// Purpose: Give one serial agent run a private encoding cache and deterministic cleanup.
// Pattern: Execution-scoped model adapter.
// Usage: Runner requests NewRunGenerator and releases it after workers and final records finish.
// Related: internal/agent/run/run.go, internal/provider/httpclient/request.go, internal/provider/protocol/encoder.go.
package httpclient

import (
	"context"

	modelapi "github.com/keonho-kim/meca/internal/agent/model"
	"github.com/keonho-kim/meca/internal/provider/protocol"
)

type runGenerator struct {
	client  *Client
	encoder protocol.Encoder
}

// NewRunGenerator shares the connection, but never cached conversation state,
// with other user requests. Its owner calls release after all Generate calls end.
func (c *Client) NewRunGenerator() (modelapi.Generator, func()) {
	g := &runGenerator{client: c}
	return g, g.encoder.Clear
}

func (g *runGenerator) Generate(ctx context.Context, request modelapi.Request, emit func(modelapi.Event) error) error {
	return g.client.generate(ctx, request, emit, false, &g.encoder)
}
