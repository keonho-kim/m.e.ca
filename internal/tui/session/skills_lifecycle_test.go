// Purpose: Release skill preview references on exit and preserve scrolling across layout completion.
// Pattern: Event-loop resource lifecycle regression tests.
// Usage: go test ./internal/tui/session -run TestSkillPreviewLifecycle.
// Related: internal/tui/session/skills.go, internal/tui/session/menu_keys.go.
package session

import (
	"strings"
	"testing"

	"github.com/gdamore/tcell/v3"
	"github.com/keonho-kim/meca/internal/skills/instruction"
	"github.com/keonho-kim/meca/internal/tui/view"
)

func TestSkillPreviewLifecycle(t *testing.T) {
	a := screenFixture(t)
	doc := &instruction.Document{Metadata: instruction.Metadata{Name: "review"}, Body: strings.Repeat("line\n", 100)}
	layout, err := view.BuildPreview(t.Context(), doc.Body, 80, false)
	if err != nil {
		t.Fatal(err)
	}
	a.page = "skills"
	a.skills.document, a.skills.preview = doc, layout
	a.previewOffset = 8
	a.receiveSkills(&skillNotice{document: doc, preview: layout, width: 80, layoutOnly: true})
	if a.previewOffset != 8 {
		t.Fatal("layout reset the current scroll")
	}
	a.menuInput.Set(":q")
	a.menuKey(tcell.NewEventKey(tcell.KeyEnter, "", 0))
	if a.skills.document != nil || a.skills.preview != nil || a.page != "chat" {
		t.Fatal("skill exit retained its document")
	}
	a.receiveSkills(&skillNotice{document: doc, preview: layout, width: 80})
	if a.skills.document != nil {
		t.Fatal("late result restored a closed preview")
	}
}

func TestSkillHintCannotReopenAfterCursorMoves(t *testing.T) {
	a := screenFixture(t)
	a.compose.editor.Set("#rev")
	a.compose.editor.Cursor = 1
	a.skills.hintBusy = true
	a.receiveSkills(&skillNotice{query: "#rev", suggestions: []instruction.Metadata{{Name: "review"}}})
	if a.skills.hintBusy || a.compose.commandOpen || len(a.compose.commandHints) != 0 {
		t.Fatal("late completion reopened hints away from the insertion boundary")
	}
}
