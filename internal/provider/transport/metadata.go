// Purpose: Retry complete bounded JSON metadata exchanges and publish only freshly decoded results.
// Pattern: Typed HTTP Adapter.
// Usage: Discovery and tokenizer callers select buffered or streaming JSON decoding.
// Related: internal/provider/transport/transport.go, internal/provider/discovery/discovery.go, internal/provider/httpclient/context.go.
package transport

import (
	"context"
	"encoding/json"
	jsonv2 "encoding/json/v2"
	"errors"
	"io"

	diagcatalog "github.com/keonho-kim/meca/internal/diagnostics/catalog"
	faults "github.com/keonho-kim/meca/internal/diagnostics/fault"
)

func ReadMetadata[T any](ctx context.Context, c *Client, method, endpoint string, payload io.Reader, limit int, target *T) error {
	return readMetadata(ctx, c, method, endpoint, payload, limit, target, func(stream *metadataStream, decoded *T) error {
		body, err := io.ReadAll(stream)
		if err != nil || stream.body.N <= 0 {
			return err
		}
		return json.Unmarshal(body, decoded)
	})
}

// ReadStreamingMetadata decodes known fields without materializing ignored values.
func ReadStreamingMetadata[T any](ctx context.Context, c *Client, method, endpoint string, payload io.Reader, limit int, target *T) error {
	return readMetadata(ctx, c, method, endpoint, payload, limit, target, func(stream *metadataStream, decoded *T) error {
		// The result stays private until the entire response succeeds, so
		// decoding can stream without the legacy whole-value validation pass.
		return jsonv2.UnmarshalRead(stream, decoded, json.DefaultOptionsV1(), json.ReportErrorsWithLegacySemantics(false))
	})
}

type metadataStream struct {
	body    io.LimitedReader
	readErr error
}

func (s *metadataStream) Read(out []byte) (int, error) {
	n, err := s.body.Read(out)
	if err != nil && err != io.EOF && s.readErr == nil {
		s.readErr = err
	}
	return n, err
}

func readMetadata[T any](ctx context.Context, c *Client, method, endpoint string, payload io.Reader, limit int, target *T, decode func(*metadataStream, *T) error) (err error) {
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
		stream := metadataStream{body: io.LimitedReader{R: response.Body, N: int64(limit) + 1}}
		var decoded T
		decodeErr := decode(&stream, &decoded)
		if decodeErr != nil && stream.readErr == nil {
			// An early JSON error must still respect the response's byte limit
			// and any transport error before it can be retried as malformed.
			_, _ = io.Copy(io.Discard, &stream)
		}
		if err := errors.Join(stream.readErr, response.Body.Close()); err != nil {
			return err
		}
		if stream.body.N <= 0 {
			return faults.Errorf(diagcatalog.TransportProviderMetadataExceedsByteLimit, limit)
		}
		if decodeErr != nil {
			return faults.Transient(faults.Errorf(diagcatalog.TransportInvalidProviderMetadata, decodeErr))
		}
		*target = decoded
		return nil
	}, nil)
}
