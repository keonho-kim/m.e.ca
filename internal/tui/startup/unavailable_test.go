// Purpose: Keep new-conversation selection usable when saved databases have unsupported formats.
// Pattern: Terminal event-queue regression test.
// Usage: go test ./internal/tui/startup -run TestUnavailableConversation.
// Related: internal/tui/startup/resume.go, internal/storage/session/session.go.
package startup

import (
	"context"
	"testing"
	"time"

	"github.com/gdamore/tcell/v3"
	"github.com/gdamore/tcell/v3/vt"
	"github.com/keonho-kim/meca/internal/config"
	sessionstate "github.com/keonho-kim/meca/internal/storage/session"
)

func TestUnavailableConversationDoesNotBlockNewSelection(t *testing.T) {
	terminal := vt.NewMockTerm(vt.MockOptSize{X: 80, Y: 12})
	screen, err := tcell.NewTerminfoScreenFromTty(terminal)
	if err != nil {
		t.Fatal(err)
	}
	if err := screen.Init(); err != nil {
		t.Fatal(err)
	}
	defer screen.Fini()
	screen.SetSize(80, 12)
	ctx, cancel := context.WithTimeout(t.Context(), 3*time.Second)
	defer cancel()
	joined := make(chan struct{})
	go func() {
		defer close(joined)
		for _, key := range []tcell.Key{tcell.KeyDown, tcell.KeyEnter, tcell.KeyUp, tcell.KeyEnter} {
			select {
			case screen.EventQ() <- tcell.NewEventKey(key, "", 0):
			case <-ctx.Done():
				return
			}
		}
	}()
	defer func() { cancel(); <-joined }()
	load := Pages([]sessionstate.Conversation{{Session: sessionstate.Session{ID: "session_AAAAAAAAAAAAAAAAAAAAAAAAAA"}, Unavailable: "unsupported saved schema"}})
	newSession := sessionstate.Session{ProjectRoot: t.TempDir()}
	selected, ok, err := Select(ctx, screen, load, newSession, config.LanguageAuto)
	if err != nil || !ok || selected != newSession || ctx.Err() != nil {
		t.Fatalf("unsupported entry blocked a new conversation: %+v %t %v", selected, ok, err)
	}
}
