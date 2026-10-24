// Purpose: Recover failed provider attempts with bounded backoff and explicit final errors.
// Pattern: Sequential retry policy.
// Usage: Generation and metadata transport share attempts without nesting retries.
// Related: internal/provider/protocol/request.go, internal/provider/transport/transport.go.
package transport

import (
	"context"
	"errors"
	"fmt"
	modelapi "github.com/keonho-kim/meca/internal/agent/model"
	"time"
)

type ConsumerError struct{ Cause error }

const maxAttempts = 3

func (e *ConsumerError) Error() string { return e.Cause.Error() }
func (e *ConsumerError) Unwrap() error { return e.Cause }

type recoveryError struct {
	attempts int
	cause    error
	detail   string
}

func (e *recoveryError) Error() string {
	return fmt.Sprintf("provider failed after %d attempts: %s", e.attempts, e.detail)
}
func (e *recoveryError) Unwrap() error { return e.cause }

func (c *Client) Attempts(ctx context.Context, run func() error, restart func(modelapi.Retry) error) error {
	for attempt := 1; attempt <= maxAttempts; attempt++ {
		if err := ctx.Err(); err != nil {
			return err
		}
		err := run()
		if err == nil {
			return nil
		}
		if ctx.Err() != nil {
			return ctx.Err()
		}
		var consumer *ConsumerError
		if errors.As(err, &consumer) {
			return consumer.Cause
		}
		if attempt == maxAttempts {
			return &recoveryError{attempt, err, c.FailureReason(err)}
		}
		// Release potentially stale connections before issuing the next request.
		c.HTTP.CloseIdleConnections()
		delay := time.Duration(attempt) * 250 * time.Millisecond
		var response *responseError
		if errors.As(err, &response) {
			delay = max(delay, response.retryAfter)
		}
		timer := time.NewTimer(delay)
		select {
		case <-ctx.Done():
			timer.Stop()
			return ctx.Err()
		case <-timer.C:
		}
		if restart != nil {
			if err := restart(modelapi.Retry{Attempt: attempt + 1, MaxAttempts: maxAttempts, Reason: c.FailureReason(err)}); err != nil {
				return err
			}
		}
	}
	return nil
}
