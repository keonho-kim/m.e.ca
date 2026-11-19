// Purpose: Share the transient observation and continuation contract across execution and storage.
// Pattern: Simple Module.
// Usage: Tool sessions own active cursors; the journal retains delivered observations.
// Related: internal/agent/run/observation_pages.go, internal/tools/browser/paging.go.
package tool

import "context"

const (
	BrowserAction   = "browser_action"
	BrowserInspect  = "browser_inspect"
	WebFetch        = "web_fetch"
	ObservationRead = `{"read":true}`
)

func IsPagedObservation(name string) bool { return name == BrowserInspect || name == WebFetch }
func IsBrowserTool(name string) bool      { return name == BrowserAction || IsPagedObservation(name) }

type ObservationCursor struct {
	Tool       string `json:"tool"`
	ResultID   string `json:"result_id"`
	Revision   string `json:"revision"`
	NextOffset int    `json:"next_offset"`
}

type ObservationCursorSource interface {
	ObservationCursors(context.Context) ([]ObservationCursor, error)
}

type webFetchDecisionKey struct{}

type webFetchDecision struct {
	bytes     int
	userInput string
}

// WithWebFetchDecision binds the page budget and the current user-authored
// instruction to the tool calls of one model decision.
func WithWebFetchDecision(ctx context.Context, bytes int, userInput string) context.Context {
	return context.WithValue(ctx, webFetchDecisionKey{}, webFetchDecision{bytes: bytes, userInput: userInput})
}

func WebFetchAllowance(ctx context.Context) (int, bool) {
	decision, ok := ctx.Value(webFetchDecisionKey{}).(webFetchDecision)
	return decision.bytes, ok
}

func WebFetchUserInput(ctx context.Context) string {
	decision, _ := ctx.Value(webFetchDecisionKey{}).(webFetchDecision)
	return decision.userInput
}
