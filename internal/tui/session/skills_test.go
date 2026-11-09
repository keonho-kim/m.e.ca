// Purpose: Verify skill browsing uses workers and previews do not submit model work.
// Pattern: Terminal behavior tests.
// Usage: go test ./internal/tui/session -run TestSkills.
// Related: internal/tui/session/skills.go, internal/tui/session/commands.go.
package session

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gdamore/tcell/v3"

	"github.com/keonho-kim/meca/internal/commands"
	"github.com/keonho-kim/meca/internal/skills"
	"time"
)

func TestSkillsBrowseAndPreview(t *testing.T) {
	a := fixture(t)
	dir := filepath.Join(a.opts.Session.ProjectRoot, ".agents/skills/review")
	if err := os.MkdirAll(dir, 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "SKILL.md"), []byte("---\nname: review\ndescription: Review source\n---\nPreview-only-instructions"), 0600); err != nil {
		t.Fatal(err)
	}
	command, err := commands.Parse("!skills")
	if err != nil {
		t.Fatal(err)
	}
	a.submitCommand(command)
	a.workers.Wait()
	select {
	case n := <-a.events:
		a.receive(n)
	default:
		t.Fatal("missing skill listing")
	}
	if a.page != "skills" || len(a.choices) == 0 || !strings.Contains(a.choices[0], "review") {
		t.Fatal(a.page, a.choices)
	}
	a.choose()
	a.workers.Wait()
	select {
	case n := <-a.events:
		a.receive(n)
	default:
		t.Fatal("missing skill preview")
	}
	a.render()
	if a.work.state.active() || a.skills.document == nil || a.skills.document.Name != "review" {
		t.Fatal("preview submitted work or lost selection")
	}
}

func TestSkillsCompletionPreservesDraftWithoutSubmission(t *testing.T) {
	a := fixture(t)
	dir := filepath.Join(a.opts.Session.ProjectRoot, ".agents/skills/review")
	if err := os.MkdirAll(dir, 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "SKILL.md"), []byte("---\nname: review\ndescription: Review source\n---\nInstructions"), 0600); err != nil {
		t.Fatal(err)
	}
	typeInput(a, "Use #rev")
	for a.skills.hintBusy {
		a.workers.Wait()
		select {
		case n := <-a.events:
			a.receive(n)
		default:
			t.Fatal("completion worker lost its result")
		}
	}
	if len(a.compose.commandHints) != 1 || a.compose.commandHints[0].Name != "#review" {
		t.Fatal(a.compose.commandHints)
	}
	a.keyEvent(tcell.NewEventKey(tcell.KeyEnter, "", 0))
	if a.compose.editor.String() != "Use #review " || a.work.state.active() {
		t.Fatal("completion submitted or replaced the draft")
	}
}

func TestSkillsCommandWaitsForOutstandingCompletion(t *testing.T) {
	a := fixture(t)
	typeInput(a, "#r")
	a.submitCommand(commands.Command{Kind: commands.Skills})
	for a.skills.busy {
		a.workers.Wait()
		select {
		case n := <-a.events:
			a.receive(n)
		default:
			t.Fatal("skill read lost its completion")
		}
	}
	if a.page != "skills" {
		t.Fatal("!skills was dropped behind autocomplete")
	}
}

func TestSkillsCompletionRefreshesSameQuery(t *testing.T) {
	a := fixture(t)
	clock := time.Unix(1000, 0)
	a.opts.Skills = skills.New(a.opts.Session.ProjectRoot, t.TempDir(), skills.Options{Redact: func(v string) string { return v }, Check: func(string) error { return nil }, Now: func() time.Time { return clock }})
	dir := filepath.Join(a.opts.Session.ProjectRoot, ".agents/skills/review")
	if err := os.MkdirAll(dir, 0700); err != nil {
		t.Fatal(err)
	}
	write := func(description string) {
		t.Helper()
		if err := os.WriteFile(filepath.Join(dir, "SKILL.md"), []byte("---\nname: review\ndescription: "+description+"\n---\nInstructions"), 0600); err != nil {
			t.Fatal(err)
		}
	}
	complete := func() {
		t.Helper()
		a.requestSkillHints("#rev")
		a.workers.Wait()
		select {
		case n := <-a.events:
			a.receive(n)
		default:
			t.Fatal("missing refreshed completion")
		}
	}
	a.compose.editor.Set("#rev")
	write("Old description")
	complete()
	write("Current description")
	clock = clock.Add(2 * time.Second)
	complete()
	if len(a.compose.commandHints) != 1 || !strings.Contains(a.compose.commandHints[0].Description, "Current description") {
		t.Fatal("same query retained stale metadata", a.compose.commandHints)
	}
}
