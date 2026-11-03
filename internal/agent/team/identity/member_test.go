// Purpose: Verify readable name uniqueness, selection and stable stored identities.
// Pattern: Pure catalog and allocation behavior tests.
// Usage: Identity tests cover the complete fixed catalog without model calls.
// Related: internal/agent/team/identity/member.go, internal/agent/team/identity/names.go.
package identity

import (
	"strings"
	"testing"
)

func TestNamesAreUniqueAndChosenWithoutReplacement(t *testing.T) {
	used := make([]string, 0, 100)
	seen := map[string]bool{}
	for range 100 {
		name, err := Choose(used)
		if err != nil || !ValidName(name) || seen[strings.ToLower(name)] {
			t.Fatal(name, err)
		}
		seen[strings.ToLower(name)] = true
		used = append(used, name)
	}
	if _, err := Choose(used); err == nil {
		t.Fatal("exhausted catalog reused a name")
	}
	if ValidName("agent-a") || ValidName("../Hana") || ValidName("") {
		t.Fatal("invalid member name accepted")
	}
}
