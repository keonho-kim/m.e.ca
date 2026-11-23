// Purpose: Verify readable name uniqueness, selection and stable stored identities.
// Pattern: Pure catalog and allocation behavior tests.
// Usage: Identity tests cover the base catalog and numbered extensions without model calls.
// Related: internal/agent/team/identity/member.go, internal/agent/team/identity/names.go.
package identity

import (
	"strings"
	"testing"
)

func TestNamesAreUniqueAndChosenWithoutReplacement(t *testing.T) {
	used := make([]string, 0, len(names)+2)
	seen := map[string]bool{}
	for range len(names) + 2 {
		name, err := Choose(used)
		if err != nil || !ValidName(name) || seen[strings.ToLower(name)] {
			t.Fatal(name, err)
		}
		seen[strings.ToLower(name)] = true
		used = append(used, name)
	}
	if used[len(names)] != "Hana-2" || used[len(names)+1] != "Ren-2" {
		t.Fatal("exhausted base catalog did not allocate numbered names", used[len(names):])
	}
	if ValidName("agent-a") || ValidName("../Hana") || ValidName("") || ValidName("Hana-1") || ValidName("Hana-02") {
		t.Fatal("invalid member name accepted")
	}
}
