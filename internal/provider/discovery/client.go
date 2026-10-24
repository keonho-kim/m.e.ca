// Purpose: Bind provider metadata queries to a caller-owned HTTP transport.
// Pattern: Metadata adapter.
// Usage: Setup supplies and closes HTTP after metadata discovery.
// Related: internal/provider/discovery/discovery.go, internal/provider/transport/client.go.
package discovery

import (
	"github.com/keonho-kim/meca/internal/provider/connection"
	"github.com/keonho-kim/meca/internal/provider/transport"
)

type Client struct {
	Connection connection.Connection
	HTTP       *transport.Client
}

func (c *Client) url(path string) string { return c.Connection.APIBase() + "/" + path }
