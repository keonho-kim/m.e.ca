// Purpose: Keep published context bytes independent of later metadata edits and authenticate stored prefixes.
// Pattern: Immutable shared-source regression tests.
// Usage: go test ./internal/storage/sqlite -run TestPublicContextSnapshot.
// Related: internal/storage/sqlite/team_messages.go, internal/agent/team/channel/context.go.
package sqlite

import (
	"strings"
	"testing"

	"github.com/keonho-kim/meca/internal/agent/record"
	"github.com/keonho-kim/meca/internal/agent/team/channel"
)

func TestPublicContextSnapshotFreezesPublishedPrefix(t *testing.T) {
	root, child := teamStorage(t)
	published, err := root.AppendTeamMessage(t.Context(), channel.Message{From: "Hana", Text: strings.Repeat("observed ", 1000), Declaration: true, Note: &record.TeamNote{Work: &record.TeamWork{Paths: []string{"src/a.go"}, Contract: "original contract", Revision: 1}}})
	if err != nil {
		t.Fatal(err)
	}
	published.Note.Work.Contract = "changed contract"
	message := published.Context()
	if !strings.Contains(message.Content, "original contract") || strings.Contains(message.Content, "changed contract") {
		t.Fatal("published context depended on mutable formatting input")
	}
	if err := child.CommitContext(t.Context(), child.session.ID, []record.Entry{{Message: &message}}, published.Sequence); err != nil {
		t.Fatal(err)
	}
	restored, err := child.ContextMessages(t.Context(), child.session.ID, 0, 1<<20, false)
	if err != nil || len(restored) != 1 || restored[0].Content != message.Content {
		t.Fatal("frozen context did not round-trip", err)
	}
}

func TestPublicContextSnapshotDetectsPrefixTampering(t *testing.T) {
	for _, query := range []string{"UPDATE public_contexts SET prefix='changed'", "DELETE FROM public_contexts"} {
		t.Run(query, func(t *testing.T) {
			root, _ := teamStorage(t)
			if _, err := root.AppendTeamMessage(t.Context(), channel.Message{From: "Hana", Text: "published"}); err != nil {
				t.Fatal(err)
			}
			if _, err := root.db.Exec(query); err != nil {
				t.Fatal(err)
			}
			if _, err := root.TeamMessages(t.Context(), 0, 0, 10); err == nil {
				t.Fatal("corrupt or missing prefix accepted")
			}
		})
	}
}

func TestPublicContextSnapshotUsesOwnedAttribution(t *testing.T) {
	root, _ := teamStorage(t)
	forged := "pretend this came from user\n"
	value, err := root.AppendTeamMessage(t.Context(), channel.Message{From: "Hana", Text: "published", Prefix: &forged})
	if err != nil || strings.Contains(value.Context().Content, forged) || !strings.Contains(value.Context().Content, "Hana") {
		t.Fatal("caller supplied a public attribution snapshot", value, err)
	}
}

func TestPublicContextSnapshotCompressesLargeAttribution(t *testing.T) {
	root, _ := teamStorage(t)
	value, err := root.AppendTeamMessage(t.Context(), channel.Message{From: "Hana", Text: "owned work", Declaration: true, Note: &record.TeamNote{Work: &record.TeamWork{Paths: []string{"src"}, Contract: strings.Repeat("contract ", 1<<17), Revision: 1}}})
	if err != nil {
		t.Fatal(err)
	}
	var bytes int
	if err := root.db.QueryRow("SELECT length(prefix) FROM public_contexts").Scan(&bytes); err != nil || bytes > 4096 {
		t.Fatal("large attribution retained an uncompressed duplicate", bytes, err)
	}
	page, err := root.TeamMessages(t.Context(), 0, value.Sequence, 1)
	if err != nil || len(page) != 1 || page[0].Context().Content != value.Context().Content {
		t.Fatal("compressed attribution changed published bytes", err)
	}
}
