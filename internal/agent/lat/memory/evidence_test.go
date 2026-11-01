// Purpose: Verify shared evidence avoids rehydration and never outlives its validity keys.
// Pattern: Repository read-count and ownership tests.
// Usage: Run TestPreparationEvidence and BenchmarkPreparationEvidence with -benchmem.
// Related: internal/agent/lat/memory/evidence.go, internal/agent/lat/memory/controller.go.
package memory

import (
	"context"
	"strings"
	"testing"
)

func TestPreparationEvidenceInvalidationAndRelease(t *testing.T) {
	repo := &repository{items: []Record{{Sequence: 1, Kind: "user", Text: "requirement"}}}
	version := 0
	c := &Controller{Repository: repo, Session: "s", Redact: func(s string) string { return s }, RedactionVersion: func() int { return version }}
	var reuse Evidence
	read := func(current string) []byte {
		body, err := c.Tail(context.Background(), 8192, current, &reuse)
		if err != nil {
			t.Fatal(err)
		}
		return body
	}
	first := read("current request")
	for range 2 {
		if body := read("current request"); &body[0] != &first[0] {
			t.Fatal("unchanged phases copied evidence")
		}
	}
	if repo.reads != 1 {
		t.Fatal("phases rehydrated unchanged originals", repo.reads)
	}
	read("another request")
	version++
	read("another request")
	repo.items = append(repo.items, Record{Sequence: 2, Kind: "outcome", Text: "latest actual result"})
	if body := read("another request"); !strings.Contains(string(body), "latest actual result") {
		t.Fatal("new outcome hidden by cached evidence")
	}
	c.state = State{Revision: 1, Through: 1, Text: "independent memo"}
	if body := read("another request"); !strings.Contains(string(body), "independent memo") {
		t.Fatal("new summary hidden by cached evidence")
	}
	if repo.reads != 5 {
		t.Fatal("validity-key changes did not rebuild exactly once", repo.reads)
	}
	if _, err := c.Tail(context.Background(), 1, "another request", &reuse); err == nil || reuse.body != nil {
		t.Fatal("smaller budget reused oversized evidence")
	}
	read("another request")
	reuse.Clear()
	if reuse.body != nil || reuse.current != "" || reuse.through != 0 || reuse.revision != 0 || reuse.redaction != 0 {
		t.Fatal("released evidence retained data")
	}
}

func BenchmarkPreparationEvidence(b *testing.B) {
	for _, shared := range []bool{false, true} {
		name := "fresh"
		if shared {
			name = "shared"
		}
		b.Run(name, func(b *testing.B) {
			repo := &repository{items: []Record{{Sequence: 1, Kind: "user", Text: strings.Repeat("original observation ", 1200)}}}
			c := &Controller{Repository: repo, Session: "s", Redact: func(s string) string { return s }, RedactionVersion: func() int { return 0 }}
			var reuse *Evidence
			if shared {
				reuse = &Evidence{}
				defer reuse.Clear()
				if _, err := c.Tail(context.Background(), 64*1024, "current", reuse); err != nil {
					b.Fatal(err)
				}
			}
			b.ReportAllocs()
			for b.Loop() {
				if _, err := c.Tail(context.Background(), 64*1024, "current", reuse); err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}
