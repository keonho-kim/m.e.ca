// Purpose: Consume one UTF-8 page at the current cursor and release its file on exhaustion.
// Pattern: Session-serialized forward-only artifact reader.
// Usage: Session.Run holds the session gate while page checks and advances the cursor.
// Related: internal/tools/browser/results.go, internal/tools/browser/session.go.
package browser

import (
	"context"

	toolapi "github.com/keonho-kim/meca/internal/agent/tool"
	diag "github.com/keonho-kim/meca/internal/diagnostics/catalog"
	"github.com/keonho-kim/meca/internal/diagnostics/fault"
)

// Cursors returns only live continuation metadata for the current mask revision.
func (s *Session) Cursors(ctx context.Context) ([]toolapi.ObservationCursor, error) {
	select {
	case s.gate <- struct{}{}:
		defer func() { <-s.gate }()
	case <-ctx.Done():
		return nil, ctx.Err()
	}
	var cursors []toolapi.ObservationCursor
	mask := s.maskVersion()
	for i := len(s.results) - 1; i >= 0; i-- {
		r := s.results[i]
		if !s.stopped && r.body.Next() > 0 && r.mask == mask {
			cursors = append(cursors, toolapi.ObservationCursor{Tool: toolapi.BrowserInspect, ResultID: r.id, Revision: r.revision, NextOffset: r.body.Next()})
		}
	}
	if mask != s.maskVersion() {
		return nil, nil
	}
	if s.documents != nil {
		cursors = append(cursors, s.documents.Cursors()...)
	}
	return cursors, nil
}

func (r storedResult) receipt(offset int, body string) Page {
	return Page{URL: r.url, ResultID: r.id, Revision: r.revision, Offset: offset, NextOffset: r.body.Next(), TotalBytes: r.body.Size(), Text: body, Untrusted: true}
}

func (r storedResult) unchanged() Page {
	page := r.receipt(0, "")
	page.Unchanged = true
	return page
}

func (s *Session) page(r Request) (Page, error) {
	for i := range s.results {
		result := &s.results[i]
		if result.id != r.ResultID {
			continue
		}
		if r.Revision != result.revision || result.mask != s.maskVersion() {
			break
		}
		limit := r.Limit
		if limit == 0 {
			limit = 8192
		}
		text, ok, err := result.body.Consume(r.Offset, limit, func() bool { return result.mask == s.maskVersion() })
		if err != nil {
			return Page{}, err
		}
		if ok {
			return result.receipt(r.Offset, text), nil
		}
		break
	}
	return Page{}, fault.Errorf(diag.BrowserResult)
}
