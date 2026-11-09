// Purpose: Preserve model-visible skill definitions when local tools enter team execution.
// Pattern: Executor adapter contract test.
// Usage: go test ./internal/tools/team -run TestTeamSkillDelivery.
// Related: internal/tools/team/executor.go, internal/tools/local/catalog/skills.go.
package team

import (
	"path/filepath"
	"reflect"
	"testing"

	toolapi "github.com/keonho-kim/meca/internal/agent/tool"
	"github.com/keonho-kim/meca/internal/tools/local"
)

func TestTeamSkillDelivery(t *testing.T) {
	project, home := t.TempDir(), t.TempDir()
	executor, err := local.NewExecutor(project, home, filepath.Join(home, ".meca"), func(s string) string { return s }, local.Options{})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := executor.Close(); err != nil {
			t.Error(err)
		}
	})
	expected := map[string]toolapi.Definition{}
	for _, d := range executor.Definitions() {
		expected[d.Name] = d
	}
	team := (&Executor{Local: executor}).Definitions()
	for _, name := range []string{"list_skills", "load_skill", "read_skill_file"} {
		found := 0
		for _, d := range team {
			if d.Name == name {
				found++
				if !reflect.DeepEqual(d, expected[name]) {
					t.Fatalf("team changed %s declaration", name)
				}
			}
		}
		if found != 1 {
			t.Fatalf("team exposed %d copies of %s", found, name)
		}
	}
}
