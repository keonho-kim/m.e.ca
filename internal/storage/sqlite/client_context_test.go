// Purpose: Store durable external team history once and restore peer references with external provenance.
// Pattern: Real public/private persistence contract test.
// Usage: go test -race ./internal/storage/sqlite -run TestExternalClientHistory.
// Related: internal/commands/server/submission_session.go, internal/storage/sqlite/team_payloads.go.
package sqlite

import (
	"strings"
	"testing"

	"github.com/keonho-kim/meca/internal/agent/record"
	"github.com/keonho-kim/meca/internal/agent/team/channel"
)

func TestExternalClientHistoryUsesOnePublicOriginal(t *testing.T) {
	root, child := teamStorage(t)
	message, err := root.AppendTeamMessage(t.Context(), channel.Message{From: channel.ClientContextAuthor, Observation: true, Text: strings.Repeat("external client history\n", 10000)})
	if err != nil {
		t.Fatal(err)
	}
	received := message.Context()
	if received.Role != "system" || received.Source.Author != channel.ClientContextAuthor || !strings.Contains(received.Content, "external notes") {
		t.Fatal("client history promoted to user instructions", received.Source)
	}
	if err := child.CommitContext(t.Context(), child.Session().ID, []record.Entry{{Message: &received}}, message.Sequence); err != nil {
		t.Fatal(err)
	}
	var codec, bytes int
	if err := child.db.QueryRow("SELECT codec,length(body) FROM payloads").Scan(&codec, &bytes); err != nil || codec != 3 || bytes > 256 {
		t.Fatal("private history cloned", codec, bytes, err)
	}
	messages, err := child.ContextMessages(t.Context(), child.Session().ID, 0, 16<<20, false, nil)
	if err != nil || len(messages) != 1 || messages[0].Content != received.Content {
		t.Fatal("external history reference lost", err)
	}
}
