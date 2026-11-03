// Purpose: Verify public provenance and immutable reference IDs round-trip independently of the room runtime.
// Pattern: Pure message-format contract tests.
// Usage: go test ./internal/agent/team/channel.
// Related: internal/agent/team/channel/context.go, internal/storage/sqlite/team_payloads.go.
package channel

import "testing"

func TestPublicContextIdentityAndProvenance(t *testing.T) {
	for _, from := range []string{"user", "Hana"} {
		message := Message{Sequence: 17, From: from, Text: "Keep the current interface."}.Context()
		sequence, ok := Sequence(message.ID)
		if !ok || sequence != 17 || IsPeer(message.ID) != (from != "user") {
			t.Fatalf("identity changed: %+v", message)
		}
		if from == "user" && (message.Role != "user" || message.Content != "Keep the current interface.") {
			t.Fatal("user instruction changed")
		}
		if from != "user" && message.Content != "Public conversation — Hana\nKeep the current interface." {
			t.Fatal("peer provenance missing")
		}
	}
	for _, id := range []string{"message_17", "peer:Hana/0", "room:user/-1", "peer:Hana/not-a-number"} {
		if _, ok := Sequence(id); ok {
			t.Fatalf("invalid context identity accepted: %s", id)
		}
	}
}
