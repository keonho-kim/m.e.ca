// Purpose: Verify Ctrl+C and forwarded macOS Meta+C exit, preserving ordinary copy variants.
// Pattern: Keyboard contract tests.
// Usage: go test ./internal/tui/input -run TestExitKey.
// Related: internal/tui/input/exit.go.
package input

import (
	"github.com/gdamore/tcell/v3"
	"runtime"
	"testing"
)

func TestExitKey(t *testing.T) {
	for _, test := range []struct {
		event *tcell.EventKey
		want  bool
	}{
		{tcell.NewEventKey(tcell.KeyCtrlC, "", 0), true},
		{tcell.NewEventKeyEx(tcell.KeyRune, "c", tcell.ModCtrl, true, tcell.KeyC, 1), true},
		{tcell.NewEventKeyEx(tcell.KeyRune, "c", tcell.ModCtrl, false, tcell.KeyC, 1), false},
		{tcell.NewEventKey(tcell.KeyRune, "c", 0), false},
		{tcell.NewEventKeyEx(tcell.KeyRune, "c", tcell.ModCtrl|tcell.ModShift, true, tcell.KeyC, 1), false},
		{tcell.NewEventKey(tcell.KeyRune, "c", tcell.ModMeta), runtime.GOOS == "darwin"},
		{tcell.NewEventKey(tcell.KeyRune, "c", tcell.ModAlt), false},
	} {
		if got := ExitKey(test.event); got != test.want {
			t.Fatalf("%s exit=%t; want %t", test.event.Name(), got, test.want)
		}
	}
}
