// Purpose: Bound complete public projections without reducing accepted work declarations.
// Pattern: Temporary SQLite regression tests.
// Usage: Team receive and metadata lookup tests run without a model.
// Related: internal/storage/sqlite/team_messages.go, internal/storage/sqlite/team_metadata.go.
package sqlite

import (
	"context"
	record "github.com/keonho-kim/meca/internal/agent/record"
	"github.com/keonho-kim/meca/internal/agent/team/channel"
	"strings"
	"testing"
)

func TestPublicPageCountsWorkMetadata(t *testing.T) {
	root, _ := teamStorage(t)
	ctx := context.Background()
	contract := strings.Repeat("x", 6<<20)
	for i := 0; i < 3; i++ {
		if _, err := root.AppendTeamMessage(ctx, channel.Message{From: "Hana", Text: "Work", Declaration: true, Note: &record.TeamNote{Work: &record.TeamWork{Paths: []string{"src"}, Contract: contract, Revision: uint64(i + 1)}}}); err != nil {
			t.Fatal(err)
		}
	}
	page, err := root.TeamMessages(ctx, 0, 0, 100)
	if err != nil || len(page) != 2 {
		t.Fatalf("page=%d err=%v", len(page), err)
	}
	size := 0
	for _, m := range page {
		size += len(m.Context().Content)
	}
	if size > 16<<20 {
		t.Fatalf("metadata escaped page budget: %d", size)
	}
	rest, err := root.TeamMessages(ctx, page[len(page)-1].Sequence, 0, 100)
	if err != nil || len(rest) != 1 {
		t.Fatal("page boundary lost remaining work", err)
	}
}

func TestPublicMetadataDoesNotHydrateBodies(t *testing.T) {
	root, _ := teamStorage(t)
	ctx := context.Background()
	value, err := root.AppendTeamMessage(ctx, channel.Message{From: "Hana", Text: strings.Repeat("x", 1<<20), Discussion: true})
	if err != nil {
		t.Fatal(err)
	}
	if _, err = root.db.Exec("DELETE FROM payloads"); err != nil {
		t.Fatal(err)
	}
	meta, err := root.TeamMessageMetadata(ctx, value.Sequence)
	if err != nil || meta.From != "Hana" || meta.Kind != "team_discussion" || meta.Bytes < 1<<20 {
		t.Fatal(meta, err)
	}
	if _, err = root.TeamMessages(ctx, 0, value.Sequence, 1); err == nil {
		t.Fatal("fixture must require body hydration")
	}
}
