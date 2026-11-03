// Purpose: Verify attributed teammate statements remain readable observations without becoming instructions or native calls.
// Pattern: Pure conversation projection regression tests.
// Usage: go test ./internal/agent/context -run TestPeer.
// Related: internal/agent/context/conversation.go, internal/agent/team/channel/context.go.
package context

import (
	"testing"

	modelapi "github.com/keonho-kim/meca/internal/agent/model"
	"github.com/keonho-kim/meca/internal/agent/team/channel"
)

func TestPeerConversationRetainsPlainAttribution(t *testing.T) {
	peer := (channel.Message{Sequence: 4, From: "agent-b", Text: "제가 api/routes를 맡겠습니다.\n저장소 인터페이스를 알려주세요."}).Context()
	projected, err := conversationMessage(peer, false)
	if err != nil || projected.Role != "user" || projected.Content != peer.Content {
		t.Fatalf("peer speech was wrapped or lost its sender: %+v %v", projected, err)
	}
	if peer.Role != "system" || !channel.IsPeer(peer.ID) {
		t.Fatal("projection changed the stored peer observation")
	}
	if projected.Reasoning != "" || len(projected.Calls) != 0 {
		t.Fatal("peer text gained executable calls or private reasoning")
	}
}

func TestPeerMarkerDoesNotReclassifyUserOrOrdinaryNotes(t *testing.T) {
	for _, original := range []modelapi.Message{
		{Role: "user", ID: "peer:agent-b/4", Content: "A real user request"},
		{Role: "system", Content: "An ordinary execution observation"},
	} {
		projected, err := conversationMessage(original, false)
		if err != nil {
			t.Fatal(err)
		}
		if original.Role == "user" && projected.Content != original.Content {
			t.Fatal("user request was changed into a teammate observation")
		}
		if original.Role == "system" && projected.Content == original.Content {
			t.Fatal("ordinary note was projected as public conversation")
		}
	}
}
