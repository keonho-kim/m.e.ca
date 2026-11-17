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
	size                    int
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
			return Page{URL: old.url, ResultID: old.id, Revision: old.revision, TotalBytes: old.size, Unchanged: true, Untrusted: true}, nil
		}
	}
	if err := memory.Resize(len(raw) * 4); err != nil {
		return Page{}, err
	}
	var object struct {
		Snapshot *string `json:"snapshot"`
		Text     *string `json:"text"`
	}
	if err := json.Unmarshal(raw, &object); err != nil {
		return Page{}, err
	}
	var text string
	if object.Snapshot != nil {
		text = *object.Snapshot
	} else if object.Text != nil {
		text = *object.Text
	} else {
		text = string(raw)
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
			return Page{URL: old.url, ResultID: old.id, Revision: old.revision, TotalBytes: old.size, Unchanged: true, Untrusted: true}, nil
		}
	}
	// Total retained results stay below 32 MiB, with at most sixteen read identities.
	var retained int
	for _, result := range s.results {
		retained += result.size
	}
	for len(s.results) > 0 && (len(s.results) >= 16 || retained+len(text) > 32<<20) {
		old := s.results[0]
		if err := os.Remove(old.path); err != nil {
			return Page{}, err
		}
		retained -= old.size
		copy(s.results, s.results[1:])
		s.results[len(s.results)-1] = storedResult{}
		s.results = s.results[:len(s.results)-1]
	}
	id := "result_" + rand.Text()
	result := storedResult{rawDigest: rawDigest, id: id, revision: revision, path: filepath.Join(s.work, id), url: address, size: len(text), mask: mask}
	file, err := os.OpenFile(result.path, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
	if err != nil {
		return Page{}, err
	}
	_, writeErr := file.WriteString(text)
	if err := errors.Join(writeErr, file.Close()); err != nil {
		os.Remove(result.path)
		return Page{}, err
	}
	if mask != s.maskVersion() {
		os.Remove(result.path)
		return Page{}, fault.Errorf(diag.BrowserResult)
	}
	s.results = append(s.results, result)
	return s.page(Request{ResultID: id, Revision: result.revision, Limit: limit})
}
func (s *Session) page(r Request) (Page, error) {
	var result *storedResult
	for i := range s.results {
		if s.results[i].id == r.ResultID {
			result = &s.results[i]
			break
		}
	}
	if result == nil || result.mask != s.maskVersion() || r.Revision == "" || result.revision != r.Revision || r.Offset > result.size {
		return Page{}, fault.Errorf(diag.BrowserResult)
	}
	limit := r.Limit
	if limit == 0 {
		limit = 8192
	}
	file, err := os.Open(result.path)
	if err != nil {
		return Page{}, err
	}
	defer file.Close()
	data := make([]byte, min(limit+4, result.size-r.Offset))
	n, err := file.ReadAt(data, int64(r.Offset))
	if err != nil && !errors.Is(err, io.EOF) {
		return Page{}, err
	}
	data = data[:n]
	if len(data) > 0 && !utf8.RuneStart(data[0]) {
		return Page{}, fault.Errorf(diag.BrowserInput, diag.Text(diag.BrowserInvalidOffset))
	}
	end := min(limit, len(data))
	for end > 0 && end < len(data) && !utf8.RuneStart(data[end]) {
		end--
	}
	next := r.Offset + end
	if next >= result.size {
		next = 0
	}
	if result.mask != s.maskVersion() {
		return Page{}, fault.Errorf(diag.BrowserResult)
	}
	return Page{URL: result.url, Text: string(data[:end]), ResultID: result.id, Revision: result.revision, Offset: r.Offset, NextOffset: next, TotalBytes: result.size, Untrusted: true}, nil
}

type readerOnly struct{ io.Reader }
