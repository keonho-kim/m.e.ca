// Purpose: Select a new or persisted conversation and revalidate existing project directories.
// Pattern: Startup presentation flow.
// Usage: Called by cmd/v with a project-filtered or global conversation query.
// Related: internal/storage/sqlite/conversations.go, internal/commands/startup/parse.go, internal/tui/view/styles.go, internal/tui/view/keymap.go.
package startup

import (
	"context"
	"fmt"
	"github.com/gdamore/tcell/v3"
	"github.com/keonho-kim/v-code/internal/commands"
	"github.com/keonho-kim/v-code/internal/commands/startup"
	sessionstate "github.com/keonho-kim/v-code/internal/storage/session"
	"github.com/keonho-kim/v-code/internal/tui/input"
	"github.com/keonho-kim/v-code/internal/tui/view"
	"github.com/rivo/uniseg"
	"strings"
)

type Conversations func(context.Context, int, int) ([]sessionstate.Conversation, error)

// Select returns newConversation with an empty ID when the first entry is chosen.
func Select(ctx context.Context, screen tcell.Screen, load Conversations, newConversation sessionstate.Session) (sessionstate.Session, bool, error) {
	events := screen.EventQ()
	offset, index := 0, -1
	status, typed := "", ""
	conversations, err := load(ctx, offset, 100)
	if err != nil {
		return sessionstate.Session{}, false, err
	}
	for {
		screen.Clear()
		w, totalHeight := screen.Size()
		shortcuts := []view.Shortcut{{Key: "^C", Label: "종료"}, {Key: "↑↓", Label: "선택"}, {Key: "Enter", Label: "선택"}, {Key: "Esc", Label: "취소"}}
		footer := view.KeymapRows(shortcuts, w, min(1, max(0, totalHeight-4)))
		h := totalHeight - footer
		view.Text(screen, 0, 0, w, "대화 선택", tcell.StyleDefault)
		first := 0
		if offset == 0 {
			first = -1
		}
		start := max(first, index-max(1, h-5)+1)
		for i := start; i < len(conversations) && i-start < h-4; i++ {
			if i == -1 {
				view.Text(screen, 0, 2+i-start, w, "새 대화", view.SelectionStyle(i == index))
				continue
			}
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
		view.Keymap(screen, shortcuts, h, w, footer)
		screen.Show()
		select {
		case <-ctx.Done():
			return sessionstate.Session{}, false, nil
		case event := <-events:
			switch e := event.(type) {
			case *tcell.EventResize:
				screen.Sync()
			case *tcell.EventKey:
				if input.ExitKey(e) {
					return sessionstate.Session{}, false, nil
				}
				switch e.Key() {
				case tcell.KeyEscape:
					return sessionstate.Session{}, false, nil
				case tcell.KeyRune:
					typed += e.Str()
					if typed == ":q" {
						return sessionstate.Session{}, false, nil
					}
					if !strings.HasPrefix(":q", typed) && len(commands.Suggestions(typed)) == 0 {
						typed = ""
					}
				case tcell.KeyUp:
					if index > first {
						index--
					} else if offset > 0 {
						offset = max(0, offset-100)
						conversations, err = load(ctx, offset, 100)
						if err != nil {
							return sessionstate.Session{}, false, err
						}
						index = max(0, len(conversations)-1)
					}
				case tcell.KeyDown:
					if index < len(conversations)-1 {
						index++
					} else if len(conversations) == 100 {
						next, queryErr := load(ctx, offset+100, 100)
						if queryErr != nil {
							return sessionstate.Session{}, false, queryErr
						}
						if len(next) > 0 {
							offset += 100
							conversations = next
							index = 0
						}
					}
				case tcell.KeyEnter:
					command, parseErr := commands.Parse(typed)
					if parseErr == nil && command.Kind == commands.Exit {
						return sessionstate.Session{}, false, nil
					}
					if typed == ":q" {
						return sessionstate.Session{}, false, nil
					}
					if index == -1 {
						return newConversation, true, nil
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
				return sessionstate.Session{}, false, fmt.Errorf("terminal: %w", e)
			}
		}
	}
}
