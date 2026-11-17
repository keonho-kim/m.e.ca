// Purpose: Verify browser observations have byte budgets, bounded retention and revision-safe Unicode pages.
// Pattern: Session artifact regression without browser processes.
// Usage: go test -race ./internal/tools/browser.
// Related: internal/tools/browser/results.go.
package browser

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"testing"
	"unicode/utf8"
)

func TestPagedObservations(t *testing.T) {
	s := New(t.TempDir(), t.TempDir(), func(text string) string { return strings.ReplaceAll(text, "secret", "masked") }, nil)
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
	if err != nil || len(files) != 16 {
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
	}, func() int { return version })
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
