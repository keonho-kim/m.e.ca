// Purpose: Verify whether skill environment requirements survive the load and revision boundary.
// Pattern: metadata contract regression test.
// Usage: go test ./internal/skills.
// Related: internal/skills/format.go, internal/skills/instruction/document.go, internal/skills/files.go.
package skills

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/keonho-kim/meca/internal/skills/instruction"
)

func TestSkillCompatibilityIsLoadedAndVersioned(t *testing.T) {
	r, project, _ := skillFixture(t)
	dir := writeSkill(t, filepath.Join(project, ".agents/skills"), "environment", "Prepare a report", "Run the bundled report script.")
	write := func(requirement string) {
		t.Helper()
		body := "---\nname: environment\ndescription: Prepare a report\ncompatibility: " + requirement + "\n---\nRun the bundled report script.\n"
		if err := os.WriteFile(filepath.Join(dir, "SKILL.md"), []byte(body), 0600); err != nil {
			t.Fatal(err)
		}
	}
	write("Requires PostgreSQL 18 and Linux")
	doc, err := r.Load(t.Context(), instruction.Selector{ID: "environment"}, true)
	if err != nil {
		t.Fatal(err)
	}
	encoded, err := json.Marshal(doc)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(encoded), "Requires PostgreSQL 18 and Linux") {
		t.Errorf("loaded instructions dropped declared compatibility: %s", encoded)
	}
	write("Requires Redis 8 and macOS")
	changed, err := r.Load(t.Context(), instruction.Selector{ID: "environment"}, true)
	if err != nil {
		t.Fatal(err)
	}
	if changed.Revision == doc.Revision {
		t.Error("compatibility-only edit did not change the instruction revision")
	}
}
