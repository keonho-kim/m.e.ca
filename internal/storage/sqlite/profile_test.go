// Purpose: Verify applied model profiles persist with serving aliases and invalid profiles leave state intact.
// Pattern: Model-selection repository tests.
// Usage: go test ./internal/storage/sqlite -run TestModelProfileSelection.
// Related: internal/storage/sqlite/selection.go, internal/provider/profile/profile.go.
package sqlite

import (
	"context"
	modelconn "github.com/keonho-kim/v-code/internal/provider/connection"
	"testing"
)

func TestModelProfileSelection(t *testing.T) {
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
	connection, err := modelconn.New(modelconn.VLLM, "")
	if err != nil {
		t.Fatal(err)
	}
	selected := modelconn.Selection{Connection: connection, Model: "local-coder", Profile: "ornith-1.5-9b"}
	if err := store.SaveSelection(ctx, session.ID, selected); err != nil {
		t.Fatal(err)
	}
	for _, invalid := range []string{"typo", "glm-5.3-flash"} {
		rejected := selected
		rejected.Profile = invalid
		if err := store.SaveSelection(ctx, session.ID, rejected); err == nil {
			t.Fatalf("invalid profile %s persisted", invalid)
		}
	}
	restored, err := store.Selection(ctx, session.ID)
	if err != nil || restored.Profile != selected.Profile || restored.Model != selected.Model {
		t.Fatalf("profile/alias lost after rejected updates: %+v %v", restored, err)
	}
}
