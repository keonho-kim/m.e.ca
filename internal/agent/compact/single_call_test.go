// Purpose: Verify repeated compaction uses one bounded request and preserves essential instructions.
// Pattern: Compaction behavior tests.
// Usage: go test ./internal/agent/compact -run TestSingle.
// Related: internal/agent/compact/controller.go, internal/agent/compact/source.go.
package compact

import (
	"context"
	"strings"
	"testing"

	modelapi "github.com/keonho-kim/v-code/internal/agent/model"
	"github.com/keonho-kim/v-code/internal/agent/prompt"
)

func TestSingleCallAfterPreviousSummary(t *testing.T) {
	repo := &repository{memory: Memory{Revision: 1, Through: 1, Markdown: "Existing requirement: preserve API."}, mark: Watermark{Tokens: 200000, Usage: 20, Timeline: 5}, records: []Record{
		{Sequence: 2, Role: "user", Text: "Keep Korean output and fix the parser."},
		{Sequence: 3, Role: "assistant", Text: strings.Repeat("large diagnostic observation\n", 10000)},
		{Sequence: 4, Role: "assistant", Text: strings.Repeat("another large observation\n", 10000)},
		{Sequence: 5, Role: "user", Text: "Also preserve cancellation."},
	}}
	c := Controller{Repository: repo, Session: "s", Model: "m", Prompt: prompt.Defaults().Compaction, Settings: Defaults(), Redact: func(s string) string { return s }}
	calls, bytes := 0, 0
	changed, err := c.Refresh(context.Background(), func(_ context.Context, r modelapi.Request, emit func(modelapi.Event) error) error {
		calls++
		for _, m := range r.Messages {
			bytes += len(m.Content)
		}
		body := r.Messages[1].Content
		for _, want := range []string{"Existing requirement", "Keep Korean", "preserve cancellation"} {
			if !strings.Contains(body, want) {
				t.Errorf("missing instruction: %s", want)
			}
		}
		return emit(modelapi.Event{Delta: "Updated memo.", Done: true})
	})
	if err != nil || !changed || calls != 1 || bytes > c.Settings.MaxInputBytes {
		t.Fatalf("changed=%t calls=%d bytes=%d err=%v", changed, calls, bytes, err)
	}
	t.Logf("raw observations=%d bytes; request=%d bytes; calls=%d", len(repo.records[1].Text)+len(repo.records[2].Text), bytes, calls)
}

func TestSingleCallPreservesMemoryWhenInstructionsCannotFit(t *testing.T) {
	prior := Memory{Revision: 2, Through: 4, UsageThrough: 10, Markdown: "Keep previous constraints"}
	repo := &repository{memory: prior, mark: Watermark{Tokens: 200000, Usage: 30, Timeline: 5}, records: []Record{{Sequence: 5, Role: "user", Text: strings.Repeat("required instruction ", 10000)}}}
	c := Controller{Repository: repo, Session: "s", Model: "m", Prompt: prompt.Defaults().Compaction, Settings: Defaults(), Redact: func(s string) string { return s }}
	calls := 0
	_, err := c.Refresh(context.Background(), func(context.Context, modelapi.Request, func(modelapi.Event) error) error { calls++; return nil })
	if err == nil || calls != 0 || repo.memory != prior || repo.installed != 0 {
		t.Fatalf("error=%v calls=%d memory=%+v", err, calls, repo.memory)
	}
}

func TestSinglePacketPrioritizesUserAndActualFailure(t *testing.T) {
	b := sourceBuffer{limit: 8192}
	redact := func(s string) string { return s }
	for _, record := range []Record{{Role: "user", Text: "Keep the public API"}, {Role: "failure", Text: "compiler failed at main.go:20"}} {
		if err := b.add(record, redact); err != nil {
			t.Fatal(err)
		}
	}
	for i := 0; i < 100; i++ {
		if err := b.add(Record{Role: "observation", Text: strings.Repeat("code\n", 1000) + string(rune('a'+i))}, redact); err != nil {
			t.Fatal(err)
		}
	}
	if err := b.add(Record{Role: "user", Text: "Preserve cancellation too"}, redact); err != nil {
		t.Fatal(err)
	}
	body := ""
	for _, entry := range b.packet() {
		body += string(entry)
	}
	for _, want := range []string{"Keep the public API", "compiler failed at main.go:20", "Preserve cancellation too"} {
		if !strings.Contains(body, want) {
			t.Errorf("lost priority source: %s", want)
		}
	}
	if b.bytes > b.limit || b.omitted == 0 {
		t.Fatalf("bytes=%d omitted=%d", b.bytes, b.omitted)
	}
}

type pagedRepository struct {
	*repository
	pages int
}

func (r *pagedRepository) MemoryRecords(ctx context.Context, session string, after, through int64) ([]Record, int64, error) {
	r.pages++
	records, cursor, err := r.repository.MemoryRecords(ctx, session, after, through)
	if len(records) > 2 {
		records = records[:2]
		cursor = records[1].Sequence
	}
	return records, cursor, err
}
func TestSingleGenerationAcrossStoragePages(t *testing.T) {
	repo := &pagedRepository{repository: &repository{mark: Watermark{Tokens: 200000, Usage: 5, Timeline: 5}}}
	for i := 1; i <= 5; i++ {
		repo.records = append(repo.records, Record{Sequence: int64(i), Role: "user", Text: strings.Repeat("x", i)})
	}
	c := Controller{Repository: repo, Session: "s", Model: "m", Prompt: prompt.Defaults().Compaction, Settings: Defaults(), Redact: func(s string) string { return s }}
	calls := 0
	changed, err := c.Refresh(context.Background(), func(_ context.Context, r modelapi.Request, emit func(modelapi.Event) error) error {
		calls++
		return emit(modelapi.Event{Delta: "memo", Done: true})
	})
	if err != nil || !changed || calls != 1 || repo.pages != 3 || repo.memory.Through != 5 {
		t.Fatalf("changed=%t calls=%d pages=%d through=%d err=%v", changed, calls, repo.pages, repo.memory.Through, err)
	}
}
