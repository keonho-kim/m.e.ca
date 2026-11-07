// Purpose: Keep idle SSE heartbeats alive while one owned cancellable I/O operation runs.
// Pattern: Joined I/O worker with a single foreground stream writer.
// Usage: Compatible streams await status, read admission and original output here.
// Related: internal/serverapi/eventstream/writer.go, internal/serverapi/openai/result.go.
package eventstream

import "context"

func Await[T any](ctx context.Context, writer *Writer, read func(context.Context) (T, error)) (T, error) {
	if writer == nil {
		return read(ctx)
	}
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	type result struct {
		value T
		err   error
	}
	ready := make(chan result, 1)
	done := make(chan struct{})
	go func() {
		defer close(done)
		value, err := read(ctx)
		ready <- result{value, err}
	}()
	defer func() { cancel(); <-done }()
	for {
		poll, stop := writer.PollContext(ctx)
		select {
		case value := <-ready:
			stop()
			if err := ctx.Err(); err != nil {
				return value.value, err
			}
			return value.value, value.err
		case <-poll.Done():
			stop()
			if err := ctx.Err(); err != nil {
				cancel()
				value := <-ready
				return value.value, err
			}
			if err := writer.Heartbeat(); err != nil {
				cancel()
				value := <-ready
				return value.value, err
			}
		}
	}
}
