// Purpose: Read bounded piped prompt input and unblock its owned descriptor on cancellation.
// Pattern: Cancellable input adapter.
// Usage: Noninteractive startup calls readPrompt before creating a session or connecting.
// Related: cmd/v/headless.go, cmd/v/input_test.go.
package main

import (
	"context"
	"fmt"
	"io"
)

func readPrompt(ctx context.Context, input io.ReadCloser) ([]byte, error) {
	closed := make(chan struct{})
	stop := context.AfterFunc(ctx, func() { input.Close(); close(closed) })
	defer func() {
		if !stop() {
			<-closed
		}
	}()
	body, err := io.ReadAll(io.LimitReader(input, 16*1024*1024+1))
	if ctx.Err() != nil {
		return nil, ctx.Err()
	}
	if err != nil {
		return nil, err
	}
	if len(body) > 16*1024*1024 {
		return nil, fmt.Errorf("stdin exceeds 16 MiB")
	}
	return body, nil
}
