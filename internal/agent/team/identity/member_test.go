// Purpose: Verify stable peer labels across alphabet boundaries and reject malformed identities.
// Pattern: Pure domain contract tests.
// Usage: go test ./internal/agent/team/identity.
// Related: internal/agent/team/identity/member.go.
package identity

import "testing"

func TestNamesRemainStableAfterFirstAlphabet(t *testing.T) {
	for index, want := range map[int]string{0: "agent-a", 25: "agent-z", 26: "agent-aa", 51: "agent-az", 52: "agent-ba", 701: "agent-zz", 702: "agent-aaa"} {
		if got := Name(index); got != want || !ValidName(got) {
			t.Fatalf("index=%d got=%s expected=%s", index, got, want)
		}
	}
	for _, id := range []string{"agent-", "agent-a/b", "Agent-a", "agent-a1", "agent-한"} {
		if ValidName(id) {
			t.Fatalf("invalid identity accepted: %s", id)
		}
	}
}
