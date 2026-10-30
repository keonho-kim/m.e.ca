// Purpose: Bind one serial metadata preparation and its last GET snapshot to a caller-owned transport.
// Pattern: Metadata adapter.
// Usage: Setup supplies and closes HTTP after metadata discovery.
// Related: internal/provider/discovery/discovery.go, internal/provider/transport/client.go.
package discovery

import (
	"encoding/json"
	"github.com/keonho-kim/meca/internal/provider/connection"
	"github.com/keonho-kim/meca/internal/provider/transport"
)

type Client struct {
	imageModel     string
	imageInput     connection.ImageInput
	ollamaName     string
	ollamaSnapshot *ollamaModel
	Connection     connection.Connection
	HTTP           *transport.Client
	lastEndpoint   string
	lastResponse   json.RawMessage
}

func (c *Client) url(path string) string { return c.Connection.APIBase() + "/" + path }
