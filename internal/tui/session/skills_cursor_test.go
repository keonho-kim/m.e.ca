// Purpose: Recover an expired skill listing without leaving a failed terminal menu.
// Pattern: Screen adapter lifecycle regression test with an injected clock.
// Usage: go test -race ./internal/tui/session -run TestSkillsExpiredCursor.
// Related: internal/tui/session/skills.go, internal/skills/snapshot.go.
package session

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/keonho-kim/meca/internal/skills"
)

func TestSkillsExpiredCursorRestartsFromCurrentCatalog(t *testing.T) {
	a := screenFixture(t)
	now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	a.opts.Skills = skills.New(a.opts.Session.ProjectRoot, a.opts.Home, skills.Options{Now: func() time.Time { return now }, Redact: func(s string) string { return s }, Check: func(string) error { return nil }})
	for _, name := range []string{"first", "second"} {
		dir := filepath.Join(a.opts.Session.ProjectRoot, ".agents", "skills", name)
		if err := os.MkdirAll(dir, 0700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dir, "SKILL.md"), []byte("---\nname: "+name+"\ndescription: Browse source\n---\nInstructions"), 0600); err != nil {
			t.Fatal(err)
		}
	}
	page, err := a.opts.Skills.List(t.Context(), "", 1)
	if err != nil || page.NextCursor == "" {
		t.Fatal(page, err)
	}
	now = now.Add(3 * time.Minute)
	a.page = "skills"
	a.loadSkillPage(page.NextCursor)
	a.workers.Wait()
	select {
	case n := <-a.events:
		a.receive(n)
	default:
		t.Fatal("missing restarted listing")
	}
	if a.skills.busy || len(a.skills.page.Items) != 2 || a.skills.page.Items[0].Name != "first" {
		t.Fatal("expired cursor left a failed menu", a.status, a.choices)
	}
}
