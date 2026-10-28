// Purpose: Build one complete file layout away from the terminal event loop.
// Pattern: Worker-owned computation with source and geometry identity.
// Usage: Session transfers successful Build results back to preview.State.
// Related: internal/tui/preview/raw.go, internal/tui/preview/markdown.go, internal/tui/session/preview_layout.go.
package preview

import "context"

type Snapshot struct {
	Path, Content  string
	Raw, Truncated bool
	Width          int
}

func Build(ctx context.Context, source Snapshot) (*Layout, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	layout := &Layout{Path: source.Path, Content: source.Content, Raw: source.Raw, Truncated: source.Truncated}
	layout.Count(source.Width)
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	return layout, nil
}
