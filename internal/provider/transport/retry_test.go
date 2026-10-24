// Purpose: Verify permanent errors return for correction while transient failures retain bounded retries.
// Pattern: Error-classification boundary tests.
// Usage: go test ./internal/provider/transport -run TestAttempts.
// Related: internal/provider/transport/retry.go, internal/provider/transport/errors.go.
package transport

import (
	"context"
	"testing"

	modelapi "github.com/keonho-kim/meca/internal/agent/model"
)

func TestAttemptsReturnsPermanentErrorsWithoutReplay(t *testing.T) {
	for _, cause := range []error{
		ParseError(401, []byte(`{"error":"invalid key"}`), ""),
		ParseError(400, []byte(`{"error":{"code":"context_length_exceeded"}}`), ""),
		&modelapi.OutputLimitError{Reason: "output token limit"},
		&modelapi.ToolCallError{Detail: "invalid arguments"},
	} {
		c := New("")
		attempts := 0
		err := c.Attempts(context.Background(), func() error { attempts++; return cause }, nil)
		c.Close()
		if err == nil || attempts != 1 {
			t.Fatalf("%T: attempts=%d error=%v", cause, attempts, err)
		}
	}
}

func TestAttemptsTransientFailureRecovers(t *testing.T) {
	for _, status := range []int{429, 503} {
		c := New("")
		attempts, resets := 0, 0
		err := c.Attempts(context.Background(), func() error {
			attempts++
			if attempts == 1 {
				return ParseError(status, nil, "")
			}
			return nil
		}, func(modelapi.Retry) error { resets++; return nil })
		c.Close()
		if err != nil || attempts != 2 || resets != 1 {
			t.Fatalf("status=%d attempts=%d resets=%d: %v", status, attempts, resets, err)
		}
	}
}
