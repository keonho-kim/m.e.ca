// Purpose: Consume one UTF-8 page at the current cursor and release its file on exhaustion.
// Pattern: Session-serialized forward-only artifact reader.
// Usage: Session.Run holds the session gate while page checks and advances the cursor.
// Related: internal/tools/browser/results.go, internal/tools/browser/session.go.
package browser

import (
	"errors"
	"io"
	"os"
	"unicode/utf8"

	diag "github.com/keonho-kim/meca/internal/diagnostics/catalog"
	"github.com/keonho-kim/meca/internal/diagnostics/fault"
)

func (r storedResult) storedBytes() int {
	if r.path == "" {
		return 0
	}
	return r.size - r.start
}

func (r storedResult) receipt(offset int, body string) Page {
	return Page{URL: r.url, ResultID: r.id, Revision: r.revision, Offset: offset, NextOffset: r.next, TotalBytes: r.size, Text: body, Untrusted: true}
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
		if result.path == "" || r.Revision != result.revision || result.mask != s.maskVersion() || r.Offset != result.next {
			break
		}
		limit := r.Limit
		if limit == 0 {
			limit = 8192
		}
		file, err := os.Open(result.path)
		if err != nil {
			return Page{}, err
		}
		data := make([]byte, min(limit+1, result.size-r.Offset))
		n, readErr := file.ReadAt(data, int64(r.Offset-result.start))
		if readErr == io.EOF && n == len(data) {
			readErr = nil
		}
		if err := errors.Join(readErr, file.Close()); err != nil {
			return Page{}, err
		}
		end := min(limit, len(data))
		for end < len(data) && end > 0 && !utf8.RuneStart(data[end]) {
			end--
		}
		if result.mask != s.maskVersion() {
			break
		}
		next := r.Offset + end
		if next == result.size {
			if err := os.Remove(result.path); err != nil {
				return Page{}, err
			}
			result.path = ""
			next = 0
		}
		result.next = next
		return result.receipt(r.Offset, string(data[:end])), nil
	}
	return Page{}, fault.Errorf(diag.BrowserResult)
}
