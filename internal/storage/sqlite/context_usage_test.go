// Purpose: Include new tool/user observations after the latest measured context without accumulating resend costs.
// Pattern: Context accounting regression test.
// Usage: go test ./internal/storage/sqlite -run TestCurrentContextIncludesTrailingResults.
// Related: internal/storage/sqlite/memory.go, internal/agent/compact/controller.go.
package sqlite

import (
	"context"
	"github.com/keonho-kim/v-code/internal/agent/compact"
	modelapi "github.com/keonho-kim/v-code/internal/agent/model"
	"strings"
	"testing"
)

func TestCurrentContextIncludesTrailingResults(t *testing.T) {
	store, err := newTestStore(t)
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	ctx := context.Background()
	id := store.Session().ID
	if err = store.AddContextTokens(ctx, id, compact.TokenCount{Amount: 100, Kind: "request"}); err != nil {
		t.Fatal(err)
	}
	if err = store.AppendMessage(ctx, id, modelapi.Message{Role: "tool", ToolCallID: "read", ToolName: "execute_command", Content: strings.Repeat("x", 300)}); err != nil {
		t.Fatal(err)
	}
	for range 2 {
		mark, err := store.MemoryWatermark(ctx, id, 0)
		if err != nil || mark.Tokens != 200 || !mark.Estimated {
			t.Fatalf("context=%+v err=%v", mark, err)
		}
	}
}
