// Purpose: Consume one UTF-8 page at the current cursor and release its file on exhaustion.
// Pattern: Session-serialized forward-only artifact reader.
// Usage: Session.Run holds the session gate while page checks and advances the cursor.
// Related: internal/tools/browser/results.go, internal/tools/browser/session.go.
package browser

import (
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"unicode/utf8"

	toolapi "github.com/keonho-kim/meca/internal/agent/tool"
	diag "github.com/keonho-kim/meca/internal/diagnostics/catalog"
	"github.com/keonho-kim/meca/internal/diagnostics/fault"
)

// Cursors returns only live continuation metadata for the current mask revision.
func (s *Session) Cursors(ctx context.Context) ([]toolapi.BrowserCursor, error) {
	select {
	case s.gate <- struct{}{}:
		defer func() { <-s.gate }()
	case <-ctx.Done():
		return nil, ctx.Err()
	}
	var cursors []toolapi.BrowserCursor
	mask := s.maskVersion()
	for i := len(s.results) - 1; i >= 0; i-- {
		r := s.results[i]
		if !s.stopped && r.next > 0 && r.mask == mask {
			cursors = append(cursors, toolapi.BrowserCursor{ResultID: r.id, Revision: r.revision, NextOffset: r.next})
		}
	}
	if mask != s.maskVersion() {
		return nil, nil
	}
	return cursors, nil
}

func (r storedResult) storedBytes() int {
	return r.size - r.start
}

// A result owns a directory of UTF-8-aligned blocks. A page reads at most one
// block, so a successful removal and its cursor advance are indivisible.
const resultBlockBytes = 128 << 10

func (r storedResult) blockPath(start int) string {
	return filepath.Join(r.path, strconv.Itoa(start))
}

func (r *storedResult) spool(text string) error {
	if err := os.Mkdir(r.path, 0700); err != nil {
		return err
	}
	for start := r.start; start < len(text); {
		end := min(start+resultBlockBytes, len(text))
		for end < len(text) && !utf8.RuneStart(text[end]) {
			end--
		}
		file, err := os.OpenFile(r.blockPath(start), os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
		if err != nil {
			return errors.Join(err, r.discard())
		}
		_, writeErr := file.WriteString(text[start:end])
		if err = errors.Join(writeErr, file.Close()); err != nil {
			return errors.Join(err, r.discard())
		}
		r.blocks = append(r.blocks, end)
		start = end
	}
	return nil
}

func (r *storedResult) discard() error {
	if r.path == "" {
		return nil
	}
	if err := os.RemoveAll(r.path); err != nil {
		return err
	}
	r.path = ""
	r.blocks = nil
	r.start, r.next = r.size, 0
	return nil
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
		if len(result.blocks) == 0 || r.Revision != result.revision || result.mask != s.maskVersion() || r.Offset != result.next {
			break
		}
		limit := r.Limit
		if limit == 0 {
			limit = 8192
		}
		path := result.blockPath(result.start)
		file, err := os.Open(path)
		if err != nil {
			return Page{}, err
		}
		blockEnd := result.blocks[0]
		data := make([]byte, min(limit+1, blockEnd-r.Offset))
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
		if next == blockEnd {
			if err := os.Remove(path); err != nil {
				return Page{}, err
			}
			result.start = blockEnd
			result.blocks = result.blocks[1:]
			if next == result.size {
				result.blocks = nil
				// A leftover empty directory can be removed on eviction/close.
				if os.Remove(result.path) == nil {
					result.path = ""
				}
				next = 0
			}
		}
		result.next = next
		return result.receipt(r.Offset, string(data[:end])), nil
	}
	return Page{}, fault.Errorf(diag.BrowserResult)
}
