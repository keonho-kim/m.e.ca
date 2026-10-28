// Purpose: Preserve request reference snapshots through queue edits, recovery and worker validation.
// Pattern: Request-value contract tests.
// Usage: go test ./internal/tui/composer.
// Related: internal/tui/composer/request.go, internal/tui/composer/queue/state.go.
package composer

import (
	"context"
	"strings"
	"testing"
)

func TestRequestReferencesAreCaptured(t *testing.T) {
	paths := []string{"a.txt", "removed.txt"}
	request := Capture("review", paths)
	paths[0] = "b.txt"
	body := request.Resolve(context.Background(), func(_ context.Context, path string) bool { return path != "removed.txt" })
	if !strings.Contains(body, "a.txt") || strings.Contains(body, "b.txt") || strings.Contains(body, "removed.txt") {
		t.Fatal(body)
	}
	if !strings.Contains(request.Body(), "removed.txt") {
		t.Fatal("validation changed the original request snapshot")
	}
}
