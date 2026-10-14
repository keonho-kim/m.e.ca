// Purpose: Retry complete bounded JSON metadata exchanges and publish only freshly decoded results.
// Pattern: Typed HTTP Adapter.
// Usage: Discovery, Thinking metadata and provider identity parsers call ReadMetadata.
// Related: internal/provider/transport/transport.go, internal/provider/discovery/discovery.go.
package transport

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
)

func ReadMetadata[T any](ctx context.Context, c *Client, method, endpoint string, payload io.Reader, limit int, target *T) (err error) {
	request, err := c.NewRequest(ctx, method, endpoint, payload)
	if err != nil {
		return err
	}
	if request.Body != nil {
		defer func() { err = errors.Join(err, request.Body.Close()) }()
		if request.GetBody == nil {
			return fmt.Errorf("provider request body must be replayable")
		}
	}
	return c.Attempts(ctx, func() error {
		attempt := request.Clone(ctx)
		if request.GetBody != nil {
			body, bodyErr := request.GetBody()
			if bodyErr != nil {
				return &ConsumerError{bodyErr}
			}
			attempt.Body = body
		}
		response, err := c.SendOnce(attempt, DefaultIdleTimeout)
		if err != nil {
			return err
		}
		body, readErr := io.ReadAll(io.LimitReader(response.Body, int64(limit)+1))
		if err := errors.Join(readErr, response.Body.Close()); err != nil {
			return err
		}
		if len(body) > limit {
			return fmt.Errorf("provider metadata exceeds %d-byte limit", limit)
		}
		var decoded T
		if err := json.Unmarshal(body, &decoded); err != nil {
			return fmt.Errorf("invalid provider metadata: %w", err)
		}
		*target = decoded
		return nil
	}, nil)
}
