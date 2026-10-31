// Purpose: Verify local UI input, approval and rendering through a virtual terminal.
// Pattern: Terminal behavior tests.
// Usage: go test ./internal/tui/session.
// Related: internal/tui/session/keys.go, internal/tui/session/render.go, internal/tui/session/tasks.go.
package session

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/gdamore/tcell/v3"
	modelconn "github.com/keonho-kim/meca/internal/provider/connection"
)

func TestAtInputRemainsLiteral(t *testing.T) {
	a := screenFixture(t)
	typeInput(a, "read @src/main.go")
	if a.editor.String() != "read @src/main.go" || a.commandOpen || a.worker.active() {
		t.Fatal("@ triggered completion")
	}
}
func TestKeyInputPreservesGraphemeStrings(t *testing.T) {
	a := screenFixture(t)
	value := "한글 a\u0301 👨‍👩‍👧‍👦"
	a.keyEvent(tcell.NewEventKey(tcell.KeyRune, value, 0))
	if a.editor.String() != value {
		t.Fatalf("key event lost Unicode text: %q", a.editor.String())
	}
}
func TestRunConsumesNativeEventQueueAndExits(t *testing.T) {
	a := fixture(t)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	finished := make(chan error, 1)
	go func() { _, err := Run(ctx, a.screen, a.opts); finished <- err }()
	for _, event := range []*tcell.EventKey{
		tcell.NewEventKey(tcell.KeyRune, ":q", 0),
		tcell.NewEventKey(tcell.KeyEnter, "", 0),
	} {
		select {
		case a.screen.EventQ() <- event:
		case <-ctx.Done():
			if err := <-finished; err != nil {
				t.Fatal(err)
			}
			t.Fatal("event queue did not accept input")
		}
	}
	if err := <-finished; err != nil {
		t.Fatal(err)
	}
	if ctx.Err() != nil {
		t.Fatal("quit did not complete before cancellation")
	}
}

func TestInputSurvivesResizeAndModelMenu(t *testing.T) {
	a := screenFixture(t)
	a.selection.Model = "chosen"
	on := true
	a.selection.Thinking = &on
	a.editor.Set("draft")
	a.render()
	line := screenLine(a.screen, 24-a.footerRows, 80)
	if strings.TrimSpace(line) != "chosen" {
		t.Fatal(line)
	}
	off := false
	a.selection.Thinking = &off
	a.render()
	if strings.Contains(screenLine(a.screen, 24-a.footerRows, 80), "추론") {
		t.Fatal("OFF is visible")
	}
	a.openModel()
	a.keyEvent(tcell.NewEventKey(tcell.KeyEscape, "", 0))
	if a.editor.String() != "draft" {
		t.Fatal("menu cleared draft")
	}
	a.screen.SetSize(32, 10)
	a.render()
	if a.editor.String() != "draft" {
		t.Fatal("resize cleared draft")
	}
}
func screenLine(screen tcell.Screen, y, width int) string {
	var b strings.Builder
	for x := 0; x < width; x++ {
		r, _, cells := screen.Get(x, y)
		b.WriteString(r)
		if cells > 1 {
			x += cells - 1
		}
	}
	return b.String()
}
func TestCancellingConnectionSelectionPreservesCurrentModel(t *testing.T) {
	a := screenFixture(t)
	a.selection.Model = "current-model"
	candidate := modelconn.Selection{Connection: modelconn.Connection{ID: "candidate"}}
	a.receive(notice{Done: true, Selection: &candidate, Models: []string{"new-model"}})
	a.keyEvent(tcell.NewEventKey(tcell.KeyEscape, "", 0))
	if a.selection.Model != "current-model" {
		t.Fatal("connection menu cleared the current model")
	}
}
