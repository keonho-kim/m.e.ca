// Purpose: Verify exit keys and !quit close the resume screen through its native event queue.
// Pattern: Terminal lifecycle tests.
// Usage: go test ./internal/tui/startup -run TestResumeExit.
// Related: internal/tui/startup/resume.go, internal/tui/input/exit.go.
package startup

import (
	"context"
	"github.com/gdamore/tcell/v3"
	"github.com/gdamore/tcell/v3/vt"
	"github.com/keonho-kim/v-code/internal/storage/sqlite"
	"path/filepath"
	"testing"
	"time"
)

func TestResumeExit(t *testing.T) {
	for _, name := range []string{"Ctrl+C", "!quit"} {
		t.Run(name, func(t *testing.T) {
			store, err := sqlite.OpenSessions(filepath.Join(t.TempDir(), "sessions"))
			if err != nil {
				t.Fatal(err)
			}
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
			ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
			finished := make(chan struct{})
			errors := make(chan error, 1)
			go func() {
				defer close(finished)
				_, selected, err := Resume(ctx, screen, store)
				if selected {
					t.Error("exit selected a conversation")
				}
				errors <- err
			}()
			defer func() { cancel(); <-finished }()
			if name == "Ctrl+C" {
				screen.EventQ() <- tcell.NewEventKey(tcell.KeyCtrlC, "", 0)
			} else {
				screen.EventQ() <- tcell.NewEventKey(tcell.KeyRune, "!quit", 0)
				screen.EventQ() <- tcell.NewEventKey(tcell.KeyEnter, "", 0)
			}
			select {
			case err := <-errors:
				if err != nil {
					t.Fatal(err)
				}
				if ctx.Err() != nil {
					t.Fatal("exit only happened after timeout")
				}
			case <-ctx.Done():
				t.Fatal("resume did not exit")
			}
		})
	}
}

func TestResumeSelectsDiscoveredSessionFile(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	sessions, err := sqlite.OpenSessions(filepath.Join(t.TempDir(), "sessions"))
	if err != nil {
		t.Fatal(err)
	}
	project, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	store, err := sessions.Create(ctx, project, project)
	if err != nil {
		t.Fatal(err)
	}
	expected := store.Session()
	if err := store.Close(); err != nil {
		t.Fatal(err)
	}
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
	finished := make(chan struct{})
	go func() {
		defer close(finished)
		select {
		case screen.EventQ() <- tcell.NewEventKey(tcell.KeyEnter, "", 0):
		case <-ctx.Done():
		}
	}()
	selected, ok, err := Resume(ctx, screen, sessions)
	<-finished
	if err != nil || !ok || selected != expected {
		t.Fatalf("selection=%+v ok=%t err=%v", selected, ok, err)
	}
	reopened, err := sessions.Open(ctx, selected.ID)
	if err != nil {
		t.Fatal(err)
	}
	if err := reopened.Close(); err != nil {
		t.Fatal(err)
	}
}
