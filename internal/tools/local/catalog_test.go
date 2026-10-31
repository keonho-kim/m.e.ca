// Purpose: Verify unknown tool names fail before an execution is prepared.
// Pattern: Tool execution boundary tests.
// Usage: go test ./internal/tools/local -run TestCatalog.
// Related: internal/tools/local/catalog/catalog.go, internal/tools/local/prepare.go.
package local

import (
	"context"
	"strings"
	"testing"

	"github.com/keonho-kim/meca/internal/policy"
)

func TestCatalogRejectsUnknownTools(t *testing.T) {
	e, _ := testExecutor(t)
	_, err := e.Prepare(context.Background(), policy.Settings{Scope: policy.System}, call("unknown_tool", `{}`))
	if err == nil || !strings.Contains(err.Error(), "unknown tool") {
		t.Fatalf("unknown tool rejection: %v", err)
	}
}
