// Purpose: Own a selected model generation client and its transport lifetime.
// Pattern: Adapter.
// Usage: Provider setup constructs the client and transfers it to generation callers.
// Related: internal/provider/connection/connection.go, internal/provider/transport/transport.go, internal/provider/protocol/request.go.

package httpclient

import (
	"fmt"
	modelconn "github.com/keonho-kim/meca/internal/provider/connection"
	"github.com/keonho-kim/meca/internal/provider/transport"
	"strings"
	"sync/atomic"
)

type Client struct {
	summaryUnsupportedModel atomic.Pointer[string]
	imageRejected           atomic.Bool
	contextLength           int
	modelInfo               *modelconn.ModelInfo
	connection              modelconn.Connection
	transport               *transport.Client
}

func NewClient(connection modelconn.Connection, key string) (*Client, error) {
	parsed, err := modelconn.New(connection.Kind, connection.Endpoint)
	if err != nil {
		return nil, err
	}
	if strings.ContainsAny(key, "\r\n\x00") || len(key) > 2048 {
		return nil, fmt.Errorf("invalid API Key")
	}
	key = strings.Trim(key, " \t")
	if key == "" && (parsed.Kind == modelconn.OpenAI || parsed.Kind == modelconn.Claude || parsed.Kind == modelconn.Gemini) {
		return nil, fmt.Errorf("this provider requires an API Key")
	}
	parsed.CredentialRef = connection.CredentialRef
	return &Client{contextLength: modelconn.DefaultContextLength, connection: parsed, transport: transport.New(key, parsed.Kind)}, nil
}

// Close releases this adapter's idle HTTP connections after its task owner has joined.
func (client *Client) Close() { client.transport.Close() }
