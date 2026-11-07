// Purpose: Reuse the last completed GET outcome during one serial connection preparation.
// Pattern: Preparation-owned bounded snapshot; transport retains retry and cancellation ownership.
// Usage: Discovery projections read a shared catalog without repeating an identical HTTP exchange.
// Related: internal/provider/discovery/client.go, internal/provider/setup/setup.go, internal/provider/transport/metadata.go.
package discovery

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"

	"github.com/keonho-kim/meca/internal/provider/transport"
)

// A Client is owned by one preparation and used serially. Only its last GET is
// retained, within the existing 16 MiB response limit; a new preparation is fresh.
func readMetadata[T any](ctx context.Context, c *Client, endpoint string, target *T) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if c.lastEndpoint == endpoint && c.lastError != nil {
		return c.lastError
	}
	if c.lastEndpoint == endpoint && c.lastResponse != nil {
		var value T
		if err := json.Unmarshal(c.lastResponse, &value); err == nil {
			*target = value
			return nil
		}
	}
	c.lastEndpoint, c.lastResponse, c.lastError = "", nil, nil
	var snapshot metadataSnapshot[T]
	if err := transport.ReadMetadata(ctx, c.HTTP, http.MethodGet, endpoint, nil, 16*1024*1024, &snapshot); err != nil {
		// Transport has already applied its retry policy. Discovery projections
		// share the same failed snapshot instead of starting that policy again.
		var shape *json.UnmarshalTypeError
		// A different projection may legitimately decode the same endpoint into
		// another shape. Its type error is not a shared transport outcome.
		if !errors.As(err, &shape) {
			c.lastEndpoint, c.lastError = endpoint, err
		}
		return err
	}
	c.lastEndpoint, c.lastResponse = endpoint, snapshot.raw
	*target = snapshot.value
	return nil
}

type metadataSnapshot[T any] struct {
	value T
	raw   json.RawMessage
}

func (s *metadataSnapshot[T]) UnmarshalJSON(body []byte) error {
	if err := json.Unmarshal(body, &s.value); err != nil {
		return err
	}
	s.raw = append(json.RawMessage(nil), body...)
	return nil
}
