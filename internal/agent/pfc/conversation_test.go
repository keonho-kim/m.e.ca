// Purpose: Verify plain replies route to chat while malformed tool entries never become executable choices.
// Pattern: Conversation parser tests.
// Usage: go test ./internal/agent/pfc -run TestConversation.
// Related: internal/agent/pfc/conversation.go.
package pfc

import "testing"

func TestConversationPlainReplyAndToolChoice(t *testing.T) {
	for _, test := range []struct {
		text string
		kind Kind
	}{
		{"Hello!", Done}, {"안녕하세요.", Done}, {"DONE\nHello!", Done},
		{"WAIT\nWhich directory?", Wait}, {`{"tool":"read_file"}`, Tool},
	} {
		decision, err := ParseConversation(test.text)
		if err != nil || decision.Kind != test.kind {
			t.Fatalf("%q: %+v %v", test.text, decision, err)
		}
	}
	for _, text := range []string{"", "GOAL: analyze\nTASK work: inspect", `{"tool":"read_file","path":"a"}`, `{"tool":`, "TOOL read_file", "CONTINUE\nI will inspect"} {
		if _, err := ParseConversation(text); err == nil {
			t.Fatalf("accepted malformed entry %q", text)
		}
	}
}
