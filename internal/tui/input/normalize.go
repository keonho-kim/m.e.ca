// Purpose: Unify Ctrl-letter events from legacy and enhanced terminal keyboard protocols.
// Pattern: Input Adapter.
// Usage: Session key routing normalizes events after bracketed-paste handling.
// Related: internal/tui/session/keys.go, internal/tui/input/keys.go.
package input

import "github.com/gdamore/tcell/v3"

func NormalizeKey(event *tcell.EventKey) *tcell.EventKey {
	modifiers := event.Modifiers() & (tcell.ModCtrl | tcell.ModAlt | tcell.ModShift | tcell.ModMeta | tcell.ModHyper)
	text := event.Str()
	if event.Key() != tcell.KeyRune || modifiers != tcell.ModCtrl || len(text) != 1 {
		return event
	}
	letter := text[0]
	if letter >= 'A' && letter <= 'Z' {
		letter += 'a' - 'A'
	}
	if letter < 'a' || letter > 'z' {
		return event
	}
	return tcell.NewEventKeyEx(tcell.KeyCtrlA+tcell.Key(letter-'a'), "", event.Modifiers(), event.Pressed(), event.Physical(), event.Repeat())
}
