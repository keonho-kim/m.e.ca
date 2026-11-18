// Purpose: Keep bounded browser observations on disk and return revision-bound UTF-8 pages.
// Pattern: Session-owned artifact window with exact page receipts.
// Usage: Browser inspection stores one redacted observation; subsequent calls use result_id and revision.
// Related: internal/tools/browser/session.go, internal/tools/browser/capture.go.
package browser

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"unicode/utf8"

	diag "github.com/keonho-kim/meca/internal/diagnostics/catalog"
	"github.com/keonho-kim/meca/internal/diagnostics/fault"
	"github.com/keonho-kim/meca/internal/processmemory"
)

type Page struct {
	Unchanged  bool   `json:"unchanged,omitempty"`
	URL        string `json:"url,omitempty"`
	Text       string `json:"text,omitempty"`
	ResultID   string `json:"result_id,omitempty"`
	Revision   string `json:"revision,omitempty"`
	Offset     int    `json:"offset,omitempty"`
	NextOffset int    `json:"next_offset,omitempty"`
	TotalBytes int    `json:"total_bytes,omitempty"`
	Untrusted  bool   `json:"untrusted,omitempty"`
	Image      []byte `json:"-"`
	memory     *processmemory.PayloadLease
}

// ToolContent keeps a correlation ID on complete observations. Only receipts
// with an unread suffix carry the revision needed to advance its cursor.
func (p Page) ToolContent() ([]byte, error) {
	if p.NextOffset == 0 {
		p.Revision = ""
		p.TotalBytes = 0
	}
	return json.Marshal(p)
}

// Close releases the image handoff reservation after the caller consumes it.
func (p *Page) Close() {
	if p.memory != nil {
		p.memory.Close()
		p.memory = nil
	}
	p.Image = nil
}

type storedResult struct {
	rawDigest               [32]byte
	id, revision, path, url string
	size, start, next       int
	mask                    int
}

func (s *Session) store(raw json.RawMessage, limit int) (Page, error) {
	memory := processmemory.NewPayloadLease()
	defer memory.Close()
	mask := s.maskVersion()
	address := s.redact(s.Current())
	rawDigest := sha256.Sum256(raw)
	for _, old := range s.results {
		if old.rawDigest == rawDigest && old.url == address && old.mask == mask {
			if mask != s.maskVersion() {
				return Page{}, fault.Errorf(diag.BrowserResult)
			}
			return old.unchanged(), nil
		}
	}
	if err := memory.Resize(len(raw) * 4); err != nil {
		return Page{}, err
	}
	var object struct {
		Snapshot *string         `json:"snapshot"`
		Text     *string         `json:"text"`
		Errors   json.RawMessage `json:"errors"`
	}
	if err := json.Unmarshal(raw, &object); err != nil {
		return Page{}, err
	}
	var text string
	if object.Snapshot != nil {
		text = *object.Snapshot
	} else if object.Text != nil {
		text = *object.Text
	} else if len(object.Errors) > 0 && object.Errors[0] == '[' {
		text = string(object.Errors)
	} else {
		return Page{}, fault.Errorf(diag.BrowserFailed, diag.Text(diag.BrowserInvalidResponse))
	}
	text = s.redact(text)
	if mask != s.maskVersion() {
		return Page{}, fault.Errorf(diag.BrowserResult)
	}
	if !utf8.ValidString(text) || len(text) > maxCapture {
		return Page{}, fault.Errorf(diag.BrowserLimit)
	}
	hash := sha256.New()
	var chunk [32 << 10]byte
	reader := strings.NewReader(text)
	if _, err := io.CopyBuffer(hash, readerOnly{reader}, chunk[:]); err != nil {
		return Page{}, err
	}
	revision := hex.EncodeToString(hash.Sum(nil))
	for i, old := range s.results {
		if old.revision == revision && old.url == address && old.mask == mask {
			if mask != s.maskVersion() {
				return Page{}, fault.Errorf(diag.BrowserResult)
			}
			s.results[i].rawDigest = rawDigest
			return old.unchanged(), nil
		}
	}
	if limit == 0 {
		limit = 2 << 10
	}
	end := min(limit, len(text))
	for end > 0 && end < len(text) && !utf8.RuneStart(text[end]) {
		end--
	}
	id := "result_" + rand.Text()
	result := storedResult{rawDigest: rawDigest, id: id, revision: revision, url: address, size: len(text), start: end, mask: mask}
	retained := len(text) - end
	for _, item := range s.results {
		retained += item.storedBytes()
	}
	for len(s.results) > 0 && (len(s.results) >= 16 || retained > 32<<20) {
		old := s.results[0]
		if old.path != "" {
			if err := os.Remove(old.path); err != nil {
				return Page{}, err
			}
		}
		retained -= old.storedBytes()
		s.results = slices.Delete(s.results, 0, 1)
	}
	if end < len(text) {
		result.path = filepath.Join(s.work, id)
		file, err := os.OpenFile(result.path, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
		if err != nil {
			return Page{}, err
		}
		_, writeErr := file.WriteString(text[end:])
		if err := errors.Join(writeErr, file.Close()); err != nil {
			return Page{}, errors.Join(err, os.Remove(result.path))
		}
		result.next = end
	}
	if mask != s.maskVersion() {
		err := fault.Errorf(diag.BrowserResult)
		if result.path != "" {
			return Page{}, errors.Join(err, os.Remove(result.path))
		}
		return Page{}, err
	}
	s.results = append(s.results, result)
	// Never pin a multi-megabyte observation through its small first page.
	body := text
	if end < len(text) {
		body = strings.Clone(text[:end])
	}
	return result.receipt(0, body), nil
}

type readerOnly struct{ io.Reader }
