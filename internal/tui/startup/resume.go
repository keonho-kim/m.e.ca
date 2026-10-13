// Purpose: Select a persisted conversation and revalidate its original project directory.
// Pattern: Startup presentation flow.
// Usage: Called by cmd/v for v resume before creating project-scoped tool adapters.
// Related: internal/storage/sqlite/selection.go, internal/commands/startup/parse.go, internal/tui/view/styles.go.
package startup

import (
	"context"
	"fmt"
	"github.com/keonho-kim/v-code/internal/tui/view"
	"strings"

	"github.com/gdamore/tcell/v3"
	"github.com/keonho-kim/v-code/internal/commands/startup"
	"github.com/keonho-kim/v-code/internal/storage/sqlite"
	"github.com/rivo/uniseg"
)

func Resume(ctx context.Context, screen tcell.Screen, store *sqlite.Store) (sqlite.Session, bool, error) {
	events := screen.EventQ()
	offset, index := 0, 0
	status, typed := "", ""
	conversations, err := store.Conversations(ctx, offset, 100)
	if err != nil {
		return sqlite.Session{}, false, err
	}
	for {
		screen.Clear()
		w, h := screen.Size()
		view.Text(screen, 0, 0, w, "Resume conversation", tcell.StyleDefault)
		if len(conversations) == 0 {
			view.Text(screen, 0, 2, w, "No conversations available", tcell.StyleDefault)
		}
		start := max(0, index-max(1, h-5)+1)
		for i := start; i < len(conversations) && i-start < h-4; i++ {
			c := conversations[i]
			style := view.SelectionStyle(i == index)
			title := strings.ReplaceAll(c.Summary, "\n", " ")
			if uniseg.StringWidth(title) > w/3 {
				r := []rune(title)
				title = string(r[:min(len(r), w/3)])
			}
			view.Text(screen, 0, 2+i-start, w, title+" │ "+c.DisplayRoot+" │ "+c.Updated, style)
		}
		if status != "" {
			view.Text(screen, 0, h-1, w, status, tcell.StyleDefault)
		}
		screen.Show()
		select {
		case <-ctx.Done():
			return sqlite.Session{}, false, nil
		case event := <-events:
			switch e := event.(type) {
			case *tcell.EventResize:
				screen.Sync()
			case *tcell.EventKey:
				switch e.Key() {
				case tcell.KeyEscape, tcell.KeyCtrlC:
					return sqlite.Session{}, false, nil
				case tcell.KeyRune:
					typed += e.Str()
					if typed == ":q" {
						return sqlite.Session{}, false, nil
					}
					if len(typed) > 2 {
						typed = ""
					}
				case tcell.KeyUp:
					if index > 0 {
						index--
					} else if offset > 0 {
						offset = max(0, offset-100)
						conversations, err = store.Conversations(ctx, offset, 100)
						if err != nil {
							return sqlite.Session{}, false, err
						}
						index = max(0, len(conversations)-1)
					}
				case tcell.KeyDown:
					if index < len(conversations)-1 {
						index++
					} else if len(conversations) == 100 {
						next, queryErr := store.Conversations(ctx, offset+100, 100)
						if queryErr != nil {
							return sqlite.Session{}, false, queryErr
						}
						if len(next) > 0 {
							offset += 100
							conversations = next
							index = 0
						}
					}
				case tcell.KeyEnter:
					if typed == ":q" {
						return sqlite.Session{}, false, nil
					}
					if len(conversations) == 0 {
						continue
					}
					selected := conversations[index].Session
					mode, pathErr := startup.Parse([]string{selected.ProjectRoot}, selected.ProjectRoot)
					if pathErr != nil {
						status = pathErr.Error()
						continue
					}
					if mode.Project != selected.ProjectRoot {
						status = "Project location changed; refusing to replace the original root"
						continue
					}
					return selected, true, nil
				}
			case *tcell.EventError:
				return sqlite.Session{}, false, fmt.Errorf("terminal: %w", e)
			}
		}
	}
}
