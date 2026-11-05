// Purpose: Preserve public visibility while distinguishing useful attention from routine speech and own actions.
// Pattern: Pure event-routing contract tests.
// Usage: go test ./internal/agent/team/channel -run TestAttention.
// Related: internal/agent/team/channel/attention.go, internal/agent/team/channel/message.go.
package channel

import (
	"testing"

	runrecord "github.com/keonho-kim/meca/internal/agent/record"
)

func TestAttentionNeverTreatsOrdinaryProgressAsARequest(t *testing.T) {
	cases := []struct {
		message Message
		member  string
		want    bool
	}{
		{Message{From: "Ren", FromStream: true, Text: "Continuing my work"}, "Hana", false},
		{Message{From: "Ren", Observation: true, Passive: true}, "Hana", false},
		{Message{From: "Ren", Observation: true}, "Hana", true},
		{Message{From: "Ren", Declaration: true}, "Ren", false},
		{Message{From: "user", Text: "New instruction"}, "Hana", true},
		{Message{From: "Ren", Discussion: true, Note: &runrecord.TeamNote{}}, "Hana", true},
		{Message{From: "Ren", Discussion: true, Note: &runrecord.TeamNote{ReplyTo: 4, ReplyAuthor: "Hana", Mentions: []string{"Arjun"}}}, "Hana", true},
		{Message{From: "Ren", Discussion: true, Note: &runrecord.TeamNote{ReplyTo: 4, ReplyAuthor: "Hana", Mentions: []string{"Arjun"}}}, "Arjun", true},
		{Message{From: "Ren", Discussion: true, Note: &runrecord.TeamNote{ReplyTo: 4, ReplyAuthor: "Hana"}}, "Mina", false},
		{Message{From: "Ren", Discussion: true, Note: &runrecord.TeamNote{ReplyTo: 4, ReplyAuthor: "Ren"}}, "Hana", true},
		{Message{From: "Ren", Discussion: true, Note: &runrecord.TeamNote{ReplyTo: 4, ReplyAuthor: "Ren"}}, "Ren", false},
		{Message{From: "Ren", Discussion: true, Note: &runrecord.TeamNote{ReplyTo: 4, ReplyAuthor: "Ren", Mentions: []string{"Hana"}}}, "Hana", true},
		{Message{From: "Ren", Discussion: true, Note: &runrecord.TeamNote{ReplyTo: 4, ReplyAuthor: "Ren", Mentions: []string{"Hana"}}}, "Mina", false},
	}
	for _, tc := range cases {
		if got := tc.message.Attention(tc.member); got != tc.want {
			t.Fatalf("%+v attention for %s = %t", tc.message, tc.member, got)
		}
	}
}

func TestAttentionMetadataCountsTowardTheSharedHistoryBudget(t *testing.T) {
	note := &runrecord.TeamNote{Work: &runrecord.TeamWork{Contract: string(make([]byte, 700*1024)), Revision: 1}}
	var history History
	history.Append(Message{Sequence: 1, From: "Hana", Text: "Own auth", Declaration: true, Note: note}, 0)
	history.Append(Message{Sequence: 2, From: "Ren", Text: "Own tests", Declaration: true, Note: note}, 1)
	if history.bytes > historyBytes || len(history.values) != 1 {
		t.Fatal("metadata retained outside the shared history budget", history.bytes, len(history.values))
	}
}
