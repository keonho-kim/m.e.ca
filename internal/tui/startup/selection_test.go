// Purpose: Verify new-conversation selection and keyboard navigation across conversation pages.
// Pattern: Virtual-terminal behavior tests.
// Usage: go test ./internal/tui/startup -run TestSelection.
// Related: internal/tui/startup/resume.go.
package startup

import (
	"context"
	"fmt"
	"github.com/gdamore/tcell/v3"
	"github.com/gdamore/tcell/v3/vt"
	sessionstate "github.com/keonho-kim/v-code/internal/storage/session"
	"path/filepath"
	"testing"
	"time"
)

func TestSelectionNewAndPagedConversations(t *testing.T) {
	for _, tc := range []struct {
		name            string
		count, down, up int
		want            string
	}{
		{"new with history", 2, 0, 0, ""}, {"empty project", 0, 0, 0, ""},
		{"first", 2, 1, 0, "0"}, {"second page", 101, 101, 0, "100"},
		{"back to new", 101, 101, 101, ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			project, err := filepath.EvalSymlinks(t.TempDir())
			if err != nil {
				t.Fatal(err)
			}
			var rows []sessionstate.Conversation
			for i := 0; i < tc.count; i++ {
				rows = append(rows, sessionstate.Conversation{Session: sessionstate.Session{ID: fmt.Sprint(i), ProjectRoot: project, DisplayRoot: project}, Summary: fmt.Sprint(i)})
			}
			load := func(_ context.Context, offset, limit int) ([]sessionstate.Conversation, error) {
				if offset >= len(rows) {
					return nil, nil
				}
				return rows[offset:min(offset+limit, len(rows))], nil
			}
			terminal := vt.NewMockTerm(vt.MockOptSize{X: 80, Y: 12})
			screen, err := tcell.NewTerminfoScreenFromTty(terminal)
			if err != nil {
				t.Fatal(err)
			}
			if err = screen.Init(); err != nil {
				t.Fatal(err)
			}
			defer screen.Fini()
			screen.SetSize(80, 12)
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			done := make(chan struct{})
			go func() {
				defer close(done)
				keys := []tcell.Key{}
				for i := 0; i < tc.down; i++ {
					keys = append(keys, tcell.KeyDown)
				}
				for i := 0; i < tc.up; i++ {
					keys = append(keys, tcell.KeyUp)
				}
				keys = append(keys, tcell.KeyEnter)
				for _, key := range keys {
					select {
					case screen.EventQ() <- tcell.NewEventKey(key, "", 0):
					case <-ctx.Done():
						return
					}
				}
			}()
			selected, ok, err := Select(ctx, screen, load, sessionstate.Session{ProjectRoot: project, DisplayRoot: project})
			cancel()
			<-done
			if err != nil || !ok || selected.ID != tc.want || selected.ProjectRoot != project {
				t.Fatalf("selected=%+v ok=%t err=%v", selected, ok, err)
			}
		})
	}
}
