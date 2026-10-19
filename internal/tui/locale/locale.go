// Purpose: Select Korean or English UI text without modifying user, provider or stored values.
// Pattern: Immutable presentation catalog.
// Usage: Terminal sessions select a Language once and pass it to presentation functions.
// Related: internal/tui/locale/system.go, internal/tui/locale/menus.go, internal/tui/locale/messages.go.
package locale

import "fmt"

type Language uint8

const (
	Korean Language = iota
	English
)

type translation struct{ id, english, korean string }

var catalog = func() map[string]translation {
	index := map[string]translation{}
	for _, entries := range [][]translation{menus, messages, controls, tools, model, approval} {
		for _, entry := range entries {
			for _, key := range []string{entry.id, entry.english, entry.korean} {
				if key != "" {
					index[key] = entry
				}
			}
		}
	}
	return index
}()

// Text accepts only application-authored UI labels. Literal content, paths,
// command output and model identifiers must not pass through this lookup.
func (l Language) Text(label string) string {
	entry, ok := catalog[label]
	if !ok {
		return label
	}
	if l == Korean {
		return entry.korean
	}
	return entry.english
}

func (l Language) Format(label string, args ...any) string {
	return fmt.Sprintf(l.Text(label), args...)
}
