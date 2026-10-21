// Purpose: Verify the conversation picker wraps hints without hiding its first selectable entry.
// Pattern: Virtual terminal layout tests.
// Usage: go test ./internal/tui/startup -run TestPickerKeymap.
// Related: internal/tui/startup/resume.go, internal/tui/view/keymap.go.
package startup

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/gdamore/tcell/v3"
	"github.com/gdamore/tcell/v3/vt"
	"github.com/keonho-kim/v-code/internal/config"
	sessionstate "github.com/keonho-kim/v-code/internal/storage/session"
)

func TestPickerKeymapWrap(t *testing.T) {
	for _, test := range []struct{ width, height, rows int }{{80, 12, 1}, {28, 12, 2}, {28, 6, 1}} {
		screen, err := tcell.NewTerminfoScreenFromTty(vt.NewMockTerm(vt.MockOptSize{X: vt.Col(test.width), Y: vt.Row(test.height)}))
		if err != nil {
			t.Fatal(err)
		}
		if err := screen.Init(); err != nil {
			t.Fatal(err)
		}
		screen.SetSize(test.width, test.height)
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		done := make(chan struct{})
		go func() {
			defer close(done)
			select {
			case screen.EventQ() <- tcell.NewEventKey(tcell.KeyEnter, "", 0):
			case <-ctx.Done():
			}
		}()
		load := func(context.Context, int, int) ([]sessionstate.Conversation, error) { return nil, nil }
		_, ok, err := Select(ctx, screen, load, sessionstate.Session{}, config.LanguageEnglish)
		cancel()
		<-done
		row := func(y int) string {
			var text strings.Builder
			for x := 0; x < test.width; x++ {
				value, _, _ := screen.Get(x, y)
				text.WriteString(value)
			}
			return text.String()
		}
		keys := ""
		for y := test.height - test.rows; y < test.height; y++ {
			keys += row(y)
		}
		choice := row(2)
		screen.Fini()
		if err != nil || !ok || !strings.Contains(choice, "New conversation") || !strings.Contains(keys, "^C") {
			t.Fatalf("%+v: choice=%q keys=%q ok=%v err=%v", test, choice, keys, ok, err)
		}
		if test.height > 6 {
			for _, key := range []string{"Esc", "Enter", "↑↓"} {
				if !strings.Contains(keys, key) {
					t.Fatalf("%+v: hint %s omitted: %q", test, key, keys)
				}
			}
		}
	}
}
