// Purpose: Preserve repeated instructions and bound source selection without front-loading large copies.
// Pattern: Selection boundary regression tests and small-path measurement.
// Usage: go test ./internal/agent/lat/memory -run TestTail -bench BenchmarkBelowThreshold.
// Related: internal/agent/lat/memory/packet.go, internal/agent/lat/memory/tail.go.
package memory

import (
	"context"
	modelapi "github.com/keonho-kim/meca/internal/agent/model"
	"strings"
	"testing"

	"github.com/keonho-kim/meca/internal/agent/prompt"
)

type sourceReads struct {
	Repository
	after []int64
}

func (r *sourceReads) LATWatermark(ctx context.Context, session string, after int64) (Mark, error) {
	r.after = append(r.after, after)
	return r.Repository.LATWatermark(ctx, session, after)
}

func TestSourceWatermarksReadOnlyNewRecordsAndResetAfterPublication(t *testing.T) {
	r := &repository{items: []Record{{Sequence: 4, Kind: "user", Text: "first"}}}
	reads := &sourceReads{Repository: r}
	c := &Controller{Repository: reads, Session: "s", Redact: func(text string) string { return text }}
	ctx := context.Background()
	prepare := func(force bool) Plan {
		p, err := c.Prepare(ctx, prompt.Defaults().LATCompaction, 262144, 2048, force)
		if err != nil {
			t.Fatal(err)
		}
		return p
	}
	prepare(false)
	r.items = append(r.items, Record{Sequence: 8, Kind: "user", Text: "second"})
	prepare(false)
	p := prepare(true)
	if err := c.Start(ctx, p, generation(func(_ context.Context, _ modelapi.Request, send func(modelapi.Event) error) error {
		return send(modelapi.Event{Delta: "summary", Done: true})
	}), func() bool { return true }); err != nil {
		t.Fatal(err)
	}
	<-c.Done()
	if _, err := c.Poll(ctx); err != nil {
		t.Fatal(err)
	}
	r.items = append(r.items, Record{Sequence: 12, Kind: "user", Text: "third"})
	prepare(false)
	if len(reads.after) != 4 || reads.after[0] != 0 || reads.after[1] != 4 || reads.after[2] != 8 || reads.after[3] != 8 || c.mark.Bytes != int64(len("third")) || c.mark.LatestUser != 12 {
		t.Fatal("source range rescanned or counts leaked across summary publication", reads.after, c.mark)
	}
}

func TestTailExcludesOnlyTheCurrentOccurrenceOfARepeatedInstruction(t *testing.T) {
	r := &repository{items: []Record{{Sequence: 1, Kind: "user", Text: "same request"}, {Sequence: 2, Kind: "outcome", Text: "actual earlier work"}, {Sequence: 3, Kind: "user", Text: "same request"}}}
	c := &Controller{Repository: r, Session: "s", Redact: func(text string) string { return text }}
	body, err := c.Tail(context.Background(), 8192, "same request")
	if err != nil || strings.Count(string(body), "same request") != 1 || !strings.Contains(string(body), "actual earlier work") {
		t.Fatal("deduplication removed a historical user instruction", string(body), err)
	}
}

func BenchmarkBelowThreshold(b *testing.B) {
	r := &repository{items: []Record{{Sequence: 1, Kind: "user", Text: "short request"}, {Sequence: 2, Kind: "outcome", Text: "actual small result"}}}
	c := &Controller{Repository: r, Session: "s", Redact: func(text string) string { return text }}
	instructions := prompt.Defaults().LATCompaction
	b.ReportAllocs()
	for b.Loop() {
		plan, err := c.Prepare(context.Background(), instructions, 262144, 2048, false)
		if err != nil || plan.Through != 0 {
			b.Fatal("small path scheduled model work", err)
		}
	}
	if r.reads != 0 || c.Active() {
		b.Fatal("small path hydrated original records or created a worker")
	}
}
