// Purpose: Own a selected model generation client and its transport lifetime.
// Pattern: Adapter.
// Usage: Provider setup constructs the client and transfers it to generation callers.
// Related: internal/provider/connection/connection.go, internal/provider/transport/transport.go, internal/provider/protocol/request.go.

package httpclient

import (
	"fmt"
	modelconn "github.com/keonho-kim/v-code/internal/provider/connection"
	"github.com/keonho-kim/v-code/internal/provider/profile"
	"github.com/keonho-kim/v-code/internal/provider/transport"
	"strings"
)

type Client struct {
	contextLength int
	modelInfo     *modelconn.ModelInfo
	profileID     string
	connection    modelconn.Connection
	transport     *transport.Client
}

func NewClient(connection modelconn.Connection, key, profileID string) (*Client, error) {
	if _, err := profile.Resolve("", profileID); err != nil {
		return nil, err
	}
	parsed, err := modelconn.New(connection.Kind, connection.Endpoint)
	if err != nil {
		return nil, err
	}
	if strings.ContainsAny(key, "\r\n") || len(key) > 16*1024 {
		return nil, fmt.Errorf("invalid API Key")
	}
	parsed.CredentialRef = connection.CredentialRef
	return &Client{contextLength: modelconn.DefaultContextLength, connection: parsed, profileID: profileID, transport: transport.New(key)}, nil
}

// Close releases this adapter's idle HTTP connections after its task owner has joined.
func (client *Client) Close() { client.transport.Close() }
