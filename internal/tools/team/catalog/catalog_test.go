// Purpose: Verify the exact team-only tool surface and owned schema copies.
// Pattern: Public tool contract tests.
// Usage: go test ./internal/tools/team -run TestTeamCatalog.
// Related: internal/tools/team/catalog/catalog.go, internal/tools/team/executor.go.
package catalog

import (
	"reflect"
	"testing"
)

func TestTeamCatalogHasOnlyMembershipAndHistoryTools(t *testing.T) {
	definitions := Definitions()
	names := make([]string, 0, len(definitions))
	for _, item := range definitions {
		names = append(names, item.Name)
		if !IsTool(item.Name) {
			t.Fatal("catalog and dispatcher differ")
		}
	}
	if !reflect.DeepEqual(names, []string{"add_team_member", "leave_team", "list_team_members", "read_team_messages", "set_work"}) {
		t.Fatal(names)
	}
	original := definitions[0].Parameters[0].Description
	definitions[0].Parameters[0].Description = "changed by caller"
	if fresh := Definitions(); fresh[0].Parameters[0].Description != original {
		t.Fatal("request mutated the private schema")
	}
}
