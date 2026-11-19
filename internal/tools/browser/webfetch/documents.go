// Purpose: Own forward-only document cursors for one browser tool session.
// Pattern: Forward-only document storage serialized by its browser owner.
// Usage: The browser session calls Prepare and serializes reading, cursor listing and cleanup.
// Related: internal/tools/observation/body.go, internal/tools/browser/webfetch/document.go.
package webfetch

import (
	"context"
	"crypto/rand"
	"errors"
	"net/url"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"unicode/utf8"

	toolapi "github.com/keonho-kim/meca/internal/agent/tool"
	diag "github.com/keonho-kim/meca/internal/diagnostics/catalog"
	"github.com/keonho-kim/meca/internal/diagnostics/fault"
	"github.com/keonho-kim/meca/internal/processmemory"
	"github.com/keonho-kim/meca/internal/tools/browser/chromium"
	"github.com/keonho-kim/meca/internal/tools/observation"
)

const FirstPageBytes = 2 << 10
const DefaultPageBytes = 8 << 10
const MaxPageBytes = 16 << 10
const maxDocuments = 8

type Input struct {
	URL, ResultID, Revision string
	Offset, Limit           *int
}

type Request struct {
	target             *url.URL
	resultID, revision string
	offset, limit      int
}

type Page struct {
	URL        string `json:"url,omitempty"`
	Title      string `json:"title,omitempty"`
	Text       string `json:"text"`
	ResultID   string `json:"result_id,omitempty"`
	Revision   string `json:"revision,omitempty"`
	Offset     int    `json:"offset,omitempty"`
	NextOffset int    `json:"next_offset,omitempty"`
	TotalBytes int    `json:"total_bytes,omitempty"`
	Untrusted  bool   `json:"untrusted"`
}

type storedDocument struct {
	id, revision string
	mask         int
	body         observation.Body
}

type Documents struct {
	client     *Client
	base, work string
	results    []storedDocument
	redact     func(string) string
	version    func() int
}

func NewDocuments(client *Client, base string, redact func(string) string, version func() int) *Documents {
	if client == nil {
		return nil
	}
	return &Documents{client: client, base: base, redact: redact, version: version}
}

func (s *Documents) Prepare(input Input) (Request, error) {
	var request Request
	if s == nil {
		return request, fault.New(diag.FailedPrecondition, diag.WebFetchDisabled)
	}
	if input.Limit != nil && (*input.Limit < utf8.UTFMax || *input.Limit > MaxPageBytes) {
		return request, fault.New(diag.InvalidArgument, diag.WebFetchInput)
	}
	if input.URL != "" {
		if input.ResultID != "" || input.Revision != "" || input.Offset != nil {
			return request, fault.New(diag.InvalidArgument, diag.WebFetchInput)
		}
		address, err := parseURL(input.URL)
		if err != nil {
			return request, err
		}
		request.target, request.limit = address, FirstPageBytes
	} else {
		if len(input.ResultID) == 0 || len(input.ResultID) > 64 || len(input.Revision) == 0 || len(input.Revision) > 64 ||
			input.Offset == nil || *input.Offset < 1 || *input.Offset >= MaxDocumentBytes {
			return request, fault.New(diag.InvalidArgument, diag.WebFetchInput)
		}
		request = Request{resultID: input.ResultID, revision: input.Revision, offset: *input.Offset, limit: DefaultPageBytes}
	}
	if input.Limit != nil {
		request.limit = *input.Limit
	}
	return request, nil
}

func (r Request) StartsDocument() bool { return r.target != nil }

func (s *Documents) Read(ctx context.Context, r Request, render chromium.Render) (Page, error) {
	if err := ctx.Err(); err != nil {
		return Page{}, err
	}
	if r.target == nil {
		return s.next(ctx, r)
	}
	return s.fetch(ctx, r, render)
}

func (s *Documents) fetch(ctx context.Context, r Request, render chromium.Render) (Page, error) {
	if _, err := pageLimit(ctx, r.limit, "", ""); err != nil {
		return Page{}, err
	}
	mask := s.maskVersion()
	if err := s.removeStale(mask); err != nil {
		return Page{}, err
	}
	if len(s.results) >= maxDocuments {
		return Page{}, fault.New(diag.ResourceExhausted, diag.WebFetchPending)
	}
	lease := processmemory.NewPayloadLease()
	defer lease.Close()
	doc, err := s.client.readDocument(ctx, r.target, lease, render)
	if err != nil {
		return Page{}, err
	}
	doc.url, doc.title, doc.text = s.redact(doc.url), s.redact(doc.title), s.redact(doc.text)
	if mask != s.maskVersion() {
		return Page{}, fault.Errorf(diag.WebFetchCursor)
	}
	if len(doc.text) > MaxDocumentBytes {
		return Page{}, fault.New(diag.ResourceExhausted, diag.WebFetchBodyLimit)
	}
	limit, err := pageLimit(ctx, r.limit, doc.url, doc.title)
	if err != nil {
		return Page{}, err
	}
	end := observation.End(doc.text, limit)
	page := Page{URL: doc.url, Title: doc.title, Text: doc.text, Untrusted: true}
	if end < len(doc.text) {
		if s.work == "" {
			s.work, err = os.MkdirTemp(s.base, "web-observations-")
			if err != nil {
				return Page{}, err
			}
		}
		item := storedDocument{id: "result_" + rand.Text(), revision: rand.Text(), mask: mask}
		item.body, err = observation.Store(filepath.Join(s.work, item.id), doc.text, end)
		if err != nil {
			return Page{}, err
		}
		if mask != s.maskVersion() || ctx.Err() != nil {
			return Page{}, errors.Join(ctx.Err(), fault.Errorf(diag.WebFetchCursor), item.body.Discard())
		}
		s.results = append(s.results, item)
		page.ResultID, page.Revision = item.id, item.revision
		page.NextOffset, page.TotalBytes = item.body.Next(), item.body.Size()
		page.Text = strings.Clone(doc.text[:end])
	}
	return page, ctx.Err()
}

func (s *Documents) next(ctx context.Context, r Request) (Page, error) {
	mask := s.maskVersion()
	for i := range s.results {
		item := &s.results[i]
		if item.id != r.resultID {
			continue
		}
		if item.revision != r.revision || item.mask != mask || item.body.Next() != r.offset {
			break
		}
		limit, err := pageLimit(ctx, r.limit, "", "")
		if err != nil {
			return Page{}, err
		}
		text, ok, err := item.body.Consume(r.offset, limit, func() bool { return mask == s.maskVersion() && ctx.Err() == nil })
		if err != nil {
			return Page{}, err
		}
		if !ok {
			break
		}
		page := Page{Text: text, Untrusted: true, Offset: r.offset, ResultID: item.id, NextOffset: item.body.Next()}
		if page.NextOffset == 0 {
			s.results = slices.Delete(s.results, i, i+1)
		} else {
			page.Revision, page.TotalBytes = item.revision, item.body.Size()
		}
		return page, nil
	}
	if err := ctx.Err(); err != nil {
		return Page{}, err
	}
	return Page{}, fault.Errorf(diag.WebFetchCursor)
}

func pageLimit(ctx context.Context, requested int, address, title string) (int, error) {
	if allowance, ok := toolapi.WebFetchAllowance(ctx); ok {
		// The envelope has fixed fields and bounded IDs. JSON can escape
		// control characters in metadata and Markdown.
		limit := (allowance - 256 - 6*(len(address)+len(title))) / 2
		if limit < utf8.UTFMax {
			return 0, fault.New(diag.ResourceExhausted, diag.WebFetchContext)
		}
		requested = min(requested, limit)
	}
	return requested, nil
}

func (s *Documents) Cursors() []toolapi.ObservationCursor {
	var cursors []toolapi.ObservationCursor
	mask := s.maskVersion()
	for _, item := range s.results {
		if item.mask == mask {
			cursors = append(cursors, toolapi.ObservationCursor{Tool: toolapi.WebFetch, ResultID: item.id, Revision: item.revision, NextOffset: item.body.Next()})
		}
	}
	if mask != s.maskVersion() {
		return nil
	}
	return cursors
}

func (s *Documents) Close() error {
	if s.work != "" {
		if err := os.RemoveAll(s.work); err != nil {
			return err
		}
		s.work = ""
	}
	s.results = nil
	return nil
}

func (s *Documents) removeStale(mask int) error {
	for i := len(s.results) - 1; i >= 0; i-- {
		if s.results[i].mask == mask {
			continue
		}
		if err := s.results[i].body.Discard(); err != nil {
			return err
		}
		s.results = slices.Delete(s.results, i, i+1)
	}
	return nil
}

func (s *Documents) maskVersion() int {
	if s.version == nil {
		return 0
	}
	return s.version()
}
