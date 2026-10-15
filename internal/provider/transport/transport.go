// Purpose: Send one authenticated HTTP attempt with idle detection and explicit status errors.
// Pattern: HTTP Adapter.
// Usage: Generation and metadata exchange owners call SendOnce within their retry loops.
// Related: internal/provider/discovery/client.go, internal/provider/transport/idle.go, internal/provider/transport/errors.go.
package transport

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"
)

func (c *Client) NewRequest(ctx context.Context, method, endpoint string, body io.Reader) (*http.Request, error) {
	request, err := http.NewRequestWithContext(ctx, method, endpoint, body)
	if err != nil {
		return nil, err
	}
	request.Header.Set("Content-Type", "application/json")
	if c.key != "" {
		request.Header.Set("Authorization", "Bearer "+c.key)
	}
	return request, nil
}

func (c *Client) SendOnce(request *http.Request, idle time.Duration) (*http.Response, error) {
	watch := watchIdle(request.Context(), idle)
	response, err := c.HTTP.Do(request.Clone(watch.ctx))
	watch.pause()
	if err != nil {
		cause := context.Cause(watch.ctx)
		watch.close()
		if cause != nil {
			return nil, cause
		}
		return nil, fmt.Errorf("provider connection failed: %w", err)
	}
	response.Body = &idleBody{ReadCloser: response.Body, watch: watch}
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		body, readErr := io.ReadAll(io.LimitReader(response.Body, 16*1024*1024+1))
		if len(body) > 16*1024*1024 {
			readErr = errors.Join(readErr, fmt.Errorf("provider error body exceeds 16 MiB"))
			body = nil
		}
		err := ParseError(response.StatusCode, body, response.Header.Get("Retry-After"))
		return nil, errors.Join(err, readErr, response.Body.Close())
	}
	return response, nil
}
