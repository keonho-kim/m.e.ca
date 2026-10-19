// Purpose: Verify cancellation releases blocked pipe input before startup can leave background work.
// Pattern: Input lifecycle regression test.
// Usage: go test ./cmd/v -run TestPipedPromptCancellation.
// Related: cmd/v/input.go.
package main

import (
	"context"
	"errors"
	"os"
	"testing"
	"time"
)

func TestPipedPromptCancellation(t *testing.T) {
	reader, writer, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	defer reader.Close()
	defer writer.Close()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	done := make(chan error, 1)
	go func() { _, err := readPrompt(ctx, reader); done <- err }()
	cancel()
	select {
	case err := <-done:
		if !errors.Is(err, context.Canceled) {
			t.Fatal(err)
		}
	case <-time.After(time.Second):
		t.Fatal("blocked stdin survived cancellation")
	}
}
