// Purpose: Check public output ordering and measure journal lookup under progress-heavy histories.
// Pattern: SQLite contract test and realistic query benchmark.
// Usage: go test ./internal/storage/sqlite -run TestResponseOutput -bench BenchmarkResponseOutput.
// Related: internal/storage/sqlite/response_output.go, internal/storage/content/text.go.
package sqlite

import (
	"fmt"
	"path/filepath"
	"strings"
	"testing"

	"github.com/keonho-kim/meca/internal/agent/model"
	"github.com/keonho-kim/meca/internal/agent/record"
)

func TestResponseOutputPreservesBoundsAndCommittedOrder(t *testing.T) {
	s, err := newTestStore(t)
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	ctx, id := t.Context(), s.Session().ID
	if err := s.AppendMessage(ctx, id, model.Message{Role: "assistant", Content: "excluded"}); err != nil {
		t.Fatal(err)
	}
	after, err := s.OutputPosition(ctx, id)
	if err != nil {
		t.Fatal(err)
	}
	var records []record.Entry
	var expected []string
	for i := range 35 {
		text := fmt.Sprintf("message %d", i)
		message := model.Message{Role: "assistant", Content: " \n" + text + "\t "}
		records = append(records, record.Entry{Message: &message})
		for range 3 {
			records = append(records, record.Entry{Event: &record.Event{Kind: "progress", Text: "update"}})
		}
		expected = append(expected, text)
	}
	incomplete := model.Message{Role: "assistant", Content: strings.Repeat("draft", 10000), Incomplete: true}
	records = append(records, record.Entry{Message: &incomplete})
	if err := s.Commit(ctx, id, records); err != nil {
		t.Fatal(err)
	}
	through, err := s.OutputPosition(ctx, id)
	if err != nil {
		t.Fatal(err)
	}
	if err := s.AppendMessage(ctx, id, model.Message{Role: "assistant", Content: "later"}); err != nil {
		t.Fatal(err)
	}
	text, err := s.ResponseOutput(ctx, id, after, through, 0, nil)
	if err != nil || text != strings.Join(expected, "\n\n") {
		t.Fatal("output boundaries or ordering changed", err, text)
	}
}

func BenchmarkResponseOutput(b *testing.B) {
	for _, shape := range []string{"progress", "messages"} {
		b.Run(shape, func(b *testing.B) {
			catalog, err := OpenSessions(filepath.Join(b.TempDir(), "sessions"))
			if err != nil {
				b.Fatal(err)
			}
			project := b.TempDir()
			s, err := catalog.Create(b.Context(), project, project)
			if err != nil {
				b.Fatal(err)
			}
			defer s.Close()
			id := s.Session().ID
			var entries []record.Entry
			for range 10000 {
				if shape == "progress" {
					entries = append(entries, record.Entry{Event: &record.Event{Kind: "progress", Text: "update"}})
				} else {
					entries = append(entries, record.Entry{Message: &model.Message{Role: "assistant", Content: "old"}})
				}
			}
			if err := s.Commit(b.Context(), id, entries); err != nil {
				b.Fatal(err)
			}
			after, err := s.OutputPosition(b.Context(), id)
			if err != nil {
				b.Fatal(err)
			}
			if shape == "progress" {
				after = 0
			}
			if err := s.AppendMessage(b.Context(), id, model.Message{Role: "assistant", Content: "answer"}); err != nil {
				b.Fatal(err)
			}
			through, err := s.OutputPosition(b.Context(), id)
			if err != nil {
				b.Fatal(err)
			}
			b.ReportAllocs()
			for b.Loop() {
				text, err := s.ResponseOutput(b.Context(), id, after, through, 0, nil)
				if err != nil || text != "answer" {
					b.Fatal(text, err)
				}
			}
		})
	}
}
