// Purpose: Bind request text to the selected paths captured when the user submits or queues it.
// Pattern: Immutable request value with worker-side reference preparation.
// Usage: Chat, steering and queued requests share Capture and Resolve.
// Related: internal/tui/session/tasks.go, internal/tui/composer/queue/state.go, internal/fileview/reader.go.
package composer

import (
	"context"
	"slices"
	"strings"
)

type Request struct {
	Text  string
	Paths []string
}

func Capture(text string, paths []string) Request {
	return Request{Text: text, Paths: slices.Clone(paths)}
}

func (r Request) Body() string { return withPaths(r.Text, r.Paths) }

// Resolve runs on the request worker; unavailable references are omitted as in normal browsing.
func (r Request) Resolve(ctx context.Context, exists func(context.Context, string) bool) string {
	if len(r.Paths) == 0 {
		return r.Text
	}
	paths := make([]string, 0, len(r.Paths))
	for _, path := range r.Paths {
		if ctx.Err() != nil {
			break
		}
		if exists(ctx, path) {
			paths = append(paths, path)
		}
	}
	return withPaths(r.Text, paths)
}

func withPaths(text string, paths []string) string {
	if len(paths) > 0 {
		text += "\nSelected path references (read only as needed):\n" + strings.Join(paths, "\n")
	}
	return text
}
