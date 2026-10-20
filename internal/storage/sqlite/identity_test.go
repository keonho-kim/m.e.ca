// Purpose: Verify bounded model identity evidence persists and invalid evidence cannot replace it.
// Pattern: Repository contract tests.
// Usage: go test ./internal/storage/sqlite -run TestIdentitySelection.
// Related: internal/storage/sqlite/selection.go, internal/provider/identity/report.go.
package sqlite

import (
	"context"
	"github.com/keonho-kim/v-code/internal/provider/connection"
	"github.com/keonho-kim/v-code/internal/provider/identity"
	"testing"
)

func TestIdentitySelection(t *testing.T) {
	store, err := newTestStore(t)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := store.Close(); err != nil {
			t.Error(err)
		}
	})
	ctx := context.Background()
	session := store.Session()
	conn, err := connection.New(connection.VLLM, "")
	if err != nil {
		t.Fatal(err)
	}
	report := &identity.Report{}
	report.Add("/v1/models.root", "original", "Qwen/Qwen3.8-27B")
	selection := connection.Selection{Connection: conn, Model: "alias", Identity: report}
	if err := store.SaveSelection(ctx, session.ID, selection); err != nil {
		t.Fatal(err)
	}
	selection.Identity = &identity.Report{Facts: []identity.Fact{{Source: "server", Field: "original", Value: "bad\x1b[31m"}}}
	if err := store.SaveSelection(ctx, session.ID, selection); err == nil {
		t.Fatal("unsafe metadata persisted")
	}
	got, err := store.Selection(ctx, session.ID)
	if err != nil || got.Identity == nil || got.Identity.Originals()[0] != "Qwen/Qwen3.8-27B" {
		t.Fatalf("selection=%+v err=%v", got, err)
	}
}
