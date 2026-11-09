// Purpose: Retry complete bounded JSON metadata exchanges and publish only freshly decoded results.
// Pattern: Typed HTTP Adapter.
// Usage: Discovery, Thinking metadata and provider identity parsers call ReadMetadata.
// Related: internal/provider/transport/transport.go, internal/provider/discovery/discovery.go.
package transport

import (
	"context"
	"encoding/json"
	"errors"

	"io"

	diagcatalog "github.com/keonho-kim/meca/internal/diagnostics/catalog"
	faults "github.com/keonho-kim/meca/internal/diagnostics/fault"
)

func ReadMetadata[T any](ctx context.Context, c *Client, method, endpoint string, payload io.Reader, limit int, target *T) (err error) {
	request, err := c.NewRequest(ctx, method, endpoint, payload)
	if err != nil {
		return err
	}
	if request.Body != nil {
		defer func() { err = errors.Join(err, request.Body.Close()) }()
		if request.GetBody == nil {
			return faults.Errorf(diagcatalog.TransportProviderRequestBodyMustBeReplayable)
		}
	}
	return c.Attempts(ctx, func() error {
		// Unlike generation, a metadata exchange has no incremental semantic
		// progress. Bound its complete attempt even if a peer trickles bytes.
		attemptCtx, cancel := context.WithTimeout(ctx, DefaultIdleTimeout)
		defer cancel()
		attempt := request.Clone(attemptCtx)
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
			return faults.Errorf(diagcatalog.TransportProviderMetadataExceedsByteLimit, limit)
		}
		var decoded T
		if err := json.Unmarshal(body, &decoded); err != nil {
			return faults.Transient(faults.Errorf(diagcatalog.TransportInvalidProviderMetadata, err))
		}
		*target = decoded
		return nil
	}, nil)
}
