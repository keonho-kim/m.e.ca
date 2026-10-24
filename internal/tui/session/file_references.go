// Purpose: Supply ordered selected paths and revalidate references attached to outgoing requests.
// Pattern: Simple Module.
// Usage: Composer projections display selected paths; new requests and steering attach valid references.
// Related: internal/tui/session/file_selection.go, internal/tui/session/render_files.go, internal/tui/session/tasks.go, internal/tui/session/steering.go.
package session

import (
	"strings"
)

func (a *app) withSelectedPaths(text string) string {
	var references []string
	for _, path := range a.selectedPaths() {
		if _, err := a.opts.Tools.Reference(a.settings, path); err == nil {
			references = append(references, path)
		}
	}
	if len(references) > 0 {
		text += "\nSelected path references (read only as needed):\n" + strings.Join(references, "\n")
	}
	return text
}
