// Purpose: Share the browser observation and continuation contract across execution and storage.
// Pattern: Simple Module.
// Usage: The browser owns active cursors; the journal retains original delivered observations.
// Related: internal/agent/run/browser_pages.go, internal/tools/browser/paging.go.
package tool

import "context"

const BrowserInspect = "browser_inspect"
const BrowserObservationRead = `{"read":true}`

type BrowserCursor struct {
	ResultID   string `json:"result_id"`
	Revision   string `json:"revision"`
	NextOffset int    `json:"next_offset"`
}

type BrowserCursorSource interface {
	BrowserCursors(context.Context) ([]BrowserCursor, error)
}
