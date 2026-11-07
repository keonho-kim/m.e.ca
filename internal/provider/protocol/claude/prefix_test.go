// Purpose: Normalize adjacent roles while preserving signed text-block and role boundaries.
// Pattern: Conversation fingerprint contract tests.
// Usage: go test ./internal/provider/protocol/claude -run TestPrefix.
// Related: internal/provider/protocol/claude/prefix.go, internal/provider/protocol/claude_request.go.
package claude

import "testing"

func TestPrefixMatchesWireGroupingAndKeepsBlockBoundaries(t *testing.T) {
	left, err := NewPrefix("rules", nil)
	if err != nil {
		t.Fatal(err)
	}
	right, err := NewPrefix("rules", []Tool{})
	if err != nil {
		t.Fatal(err)
	}
	for _, text := range []string{"one", "two"} {
		if err := left.Append("user", []Block{{Type: "text", Text: text}}); err != nil {
			t.Fatal(err)
		}
	}
	if err := right.Append("user", []Block{{Type: "text", Text: "one"}, {Type: "text", Text: "two"}}); err != nil {
		t.Fatal(err)
	}
	if left.Value() != right.Value() {
		t.Fatal("identical wire grouping had different prefixes")
	}
	coalesced, _ := NewPrefix("rules", nil)
	if err := coalesced.Append("user", []Block{{Type: "text", Text: "onetwo"}}); err != nil {
		t.Fatal(err)
	}
	if coalesced.Value() == left.Value() {
		t.Fatal("text block boundaries were discarded")
	}
	other, _ := NewPrefix("rules", nil)
	if err := other.Append("assistant", []Block{{Type: "text", Text: "one"}, {Type: "text", Text: "two"}}); err != nil {
		t.Fatal(err)
	}
	if other.Value() == left.Value() {
		t.Fatal("role boundaries were discarded")
	}
}
