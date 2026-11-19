// Purpose: Verify browser observations have byte budgets, bounded retention and revision-safe Unicode pages.
// Pattern: Session artifact regression without browser processes.
// Usage: go test -race ./internal/tools/browser.
// Related: internal/tools/browser/results.go.
package browser

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/keonho-kim/meca/internal/tools/observation"
)

func TestBrowserPaginationReclaimsConsumedBlocks(t *testing.T) {
	s := New("", "", func(text string) string { return text }, nil, nil)
	s.work = t.TempDir()
	text := strings.Repeat("x界😀", 96<<10)
	raw, _ := json.Marshal(map[string]string{"text": text})
	first, err := s.store(raw, 0)
	if err != nil {
		t.Fatal(err)
	}
	out := strings.Builder{}
	out.WriteString(first.Text)
	req := Request{ResultID: first.ResultID, Revision: first.Revision, Offset: first.NextOffset, Limit: 16381}
	for req.Offset < len(text)*9/10 {
		old := req
		p, err := s.Run(t.Context(), req, "")
		if err != nil || !utf8.ValidString(p.Text) || p.NextOffset <= req.Offset {
			t.Fatal("bad page following a block boundary", p.NextOffset, req.Offset, err)
		}
		out.WriteString(p.Text)
		req.Offset = p.NextOffset
		if _, err := s.Run(t.Context(), old, ""); err == nil {
			t.Fatal("page replay succeeded")
		}
	}
	result := s.results[0]
	if len(text)-result.body.Retained() < req.Offset-observation.BlockBytes || result.body.Retained() > len(text)-req.Offset+observation.BlockBytes {
		t.Fatal("consumed bytes retained", result.body.Retained())
	}
	if _, err := os.Stat(filepath.Join(s.work, result.id, strconv.Itoa(first.NextOffset))); !os.IsNotExist(err) {
		t.Fatal("the first disk block still exists", err)
	}
	cursors, err := s.Cursors(t.Context())
	if err != nil || len(cursors) != 1 || cursors[0].NextOffset != req.Offset {
		t.Fatal("cursor diverged from committed page", cursors, err)
	}
	for req.Offset != 0 {
		p, err := s.Run(t.Context(), req, "")
		if err != nil {
			t.Fatal(err)
		}
		out.WriteString(p.Text)
		req.Offset = p.NextOffset
	}
	cursors, err = s.Cursors(t.Context())
	if out.String() != text || err != nil || len(cursors) != 0 {
		t.Fatal("final page or cursor lifetime differs", len(cursors), err)
	}
}

func TestPagedObservations(t *testing.T) {
	s := New(t.TempDir(), t.TempDir(), func(text string) string { return strings.ReplaceAll(text, "secret", "masked") }, nil, nil)
	s.work = t.TempDir()
	s.setAddress("https://original.example/")
	raw, _ := json.Marshal(map[string]string{"text": strings.Repeat("界😀secret", 2000)})
	first, err := s.store(raw, 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(first.Text) > 8192 || !utf8.ValidString(first.Text) || strings.Contains(first.Text, "secret") || first.NextOffset == 0 {
		t.Fatal("invalid first page", first)
	}
	same, err := s.store(raw, 0)
	if err != nil || !same.Unchanged || same.Text != "" || same.ResultID != first.ResultID {
		t.Fatal("repeated observation copied its body", same, err)
	}
	s.setAddress("https://next.example/")
	out := first.Text
	offset := first.NextOffset
	for offset != 0 {
		page, err := s.page(Request{ResultID: first.ResultID, Revision: first.Revision, Offset: offset, Limit: 31})
		if err != nil {
			t.Fatal(err)
		}
		if page.URL != "https://original.example/" {
			t.Fatal("result source changed after navigation", page.URL)
		}
		out += page.Text
		offset = page.NextOffset
	}
	if out != strings.Repeat("界😀masked", 2000) {
		t.Fatal("paged body changed")
	}
	if _, err := s.page(Request{ResultID: first.ResultID, Revision: "stale"}); err == nil {
		t.Fatal("stale revision accepted")
	}
	if _, err := s.page(Request{ResultID: first.ResultID, Revision: first.Revision, Offset: 1}); err == nil {
		t.Fatal("split Unicode accepted")
	}
	for i := range 20 {
		if _, err := s.store(json.RawMessage(fmt.Sprintf(`{"text":"new %d"}`, i)), 0); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := s.page(Request{ResultID: first.ResultID, Revision: first.Revision}); err == nil {
		t.Fatal("expired result retained")
	}
	files, err := os.ReadDir(s.work)
	if err != nil || len(files) != 0 || len(s.results) != 16 {
		t.Fatal("unbounded artifacts", len(files), err)
	}
}

func TestCredentialRevisionExpiresObservation(t *testing.T) {
	version := 0
	s := New(t.TempDir(), t.TempDir(), func(text string) string {
		if version > 0 {
			return strings.ReplaceAll(text, "secret", "masked")
		}
		return text
	}, func() int { return version }, nil)

	s.work = t.TempDir()
	first, err := s.store(json.RawMessage(`{"text":"secret"}`), 0)
	if err != nil {
		t.Fatal(err)
	}
	version++
	if _, err := s.page(Request{ResultID: first.ResultID, Revision: first.Revision}); err == nil {
		t.Fatal("old credential projection was delivered")
	}
	next, err := s.store(json.RawMessage(`{"text":"secret"}`), 0)
	if err != nil || next.Text != "masked" || next.ResultID == first.ResultID {
		t.Fatal(next, err)
	}
}

func TestObservationDeliveryBudget(t *testing.T) {
	s := New("", "", func(text string) string { return text }, nil, nil)
	s.work = t.TempDir()
	s.setAddress("https://local.example/")
	store := func(body string, limit int) Page {
		t.Helper()
		raw, err := json.Marshal(map[string]any{"snapshot": body, "refs": map[string]string{"fixture-internal": "fixture-only"}})
		if err != nil {
			t.Fatal(err)
		}
		p, err := s.store(raw, limit)
		if err != nil {
			t.Fatal(err)
		}
		return p
	}
	short := store("Ready", 0)
	encoded, err := short.ToolContent()
	if err != nil {
		t.Fatal(err)
	}
	full, err := json.Marshal(short)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("complete observation: %d bytes with full metadata, %d bytes delivered", len(full), len(encoded))
	fields := map[string]any{}
	if err := json.Unmarshal(encoded, &fields); err != nil {
		t.Fatal(err)
	}
	if fields["text"] != "Ready" || fields["url"] != "https://local.example/" || fields["untrusted"] != true || fields["result_id"] != short.ResultID || fields["revision"] != nil || fields["total_bytes"] != nil || strings.Contains(string(encoded), "fixture-only") {
		t.Fatal(string(encoded))
	}
	long := strings.Repeat("界", 5000)
	first := store(long, 0)
	if len(first.Text) > 2048 || first.NextOffset != len(first.Text) || first.NextOffset == 0 {
		t.Fatal("invalid first page", len(first.Text), first.NextOffset)
	}
	unchanged := store(long, 0)
	if !unchanged.Unchanged || unchanged.ResultID != first.ResultID || unchanged.Revision != first.Revision {
		t.Fatal("lost earlier observation")
	}
	assembled := first.Text
	for page := first; page.NextOffset != 0; {
		var err error
		page, err = s.page(Request{ResultID: page.ResultID, Revision: page.Revision, Offset: page.NextOffset})
		if err != nil {
			t.Fatal(err)
		}
		if len(page.Text) > 8192 {
			t.Fatal("unbounded continuation")
		}
		assembled += page.Text
	}
	if assembled != long {
		t.Fatal("missing original text")
	}
	explicit := store(long+"!", 4096)
	if len(explicit.Text) <= 2048 || len(explicit.Text) > 4096 {
		t.Fatal("ignored explicit first-page size")
	}
}

func TestObserveErrorsExcludesHelperMetadata(t *testing.T) {
	s := New("", "", func(text string) string { return text }, nil, nil)
	s.work = t.TempDir()
	errors := `[{"text":"bad","url":"fixture.js","line":3,"column":2}]`
	page, err := s.store(json.RawMessage(`{"errors":`+errors+`,"internal":"fixture-secret"}`), 0)
	if err != nil || page.Text != errors {
		t.Fatal(page, err)
	}
	if _, err := s.store(json.RawMessage(`{"internal":"fixture-secret"}`), 0); err == nil {
		t.Fatal("unexpected helper data published")
	}
}

func TestConsumePageOnce(t *testing.T) {
	s := New("", "", func(v string) string { return v }, nil, nil)
	s.work = t.TempDir()
	raw := json.RawMessage(`{"text":"abcd世界xy"}`)
	first, err := s.store(raw, 4)
	if err != nil || first.Text != "abcd" {
		t.Fatal(first, err)
	}
	read := Request{ResultID: first.ResultID, Revision: first.Revision, Limit: 4}
	for _, bad := range []int{0, 7} {
		read.Offset = bad
		if _, err := s.page(read); err == nil {
			t.Fatal("accepted consumed or skipped offset", bad)
		}
	}
	read.Offset = first.NextOffset
	next, err := s.page(read)
	if err != nil || next.Text != "世" {
		t.Fatal(next, err)
	}
	if _, err := s.page(read); err == nil {
		t.Fatal("page delivered twice")
	}
	same, err := s.store(raw, 4)
	if err != nil || !same.Unchanged || same.NextOffset != next.NextOffset {
		t.Fatal("duplicate lost live cursor", same, err)
	}
	read.Offset = next.NextOffset
	read.Limit = 8
	last, err := s.page(read)
	if err != nil || last.NextOffset != 0 || last.Text != "界xy" {
		t.Fatal(last, err)
	}
	if _, err := s.page(read); err == nil {
		t.Fatal("read exhausted result")
	}
	files, err := os.ReadDir(s.work)
	if err != nil || len(files) != 0 {
		t.Fatal("retained body at EOF", files, err)
	}
	same, err = s.store(raw, 4)
	if err != nil || !same.Unchanged || same.NextOffset != 0 {
		t.Fatal(same, err)
	}
}

func TestConcurrentContinuationHasOneConsumer(t *testing.T) {
	s := New("", "", func(text string) string { return text }, nil, nil)
	s.work = t.TempDir()
	first, err := s.store(json.RawMessage(`{"text":"abcdefgh"}`), 4)
	if err != nil {
		t.Fatal(err)
	}
	r := Request{ResultID: first.ResultID, Revision: first.Revision, Offset: first.NextOffset}
	start := make(chan struct{})
	results := make(chan error, 2)
	for range 2 {
		go func() { <-start; _, err := s.Run(t.Context(), r, ""); results <- err }()
	}
	close(start)
	a, b := <-results, <-results
	if (a == nil) == (b == nil) {
		t.Fatal("expected exactly one delivery", a, b)
	}
}
