// Purpose: Verify shared public windows preserve receiver isolation, gaps, eviction and cancellation.
// Pattern: Deterministic repository-call and resource-bound tests.
// Usage: go test -race ./internal/agent/team/channel -run TestPublicHistory.
// Related: internal/agent/team/channel/history.go, internal/agent/team/inbox.go.
package channel

import (
	"context"
	"errors"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/keonho-kim/meca/internal/agent/media"
	runrecord "github.com/keonho-kim/meca/internal/agent/record"
	"github.com/keonho-kim/meca/internal/agent/team/source"
)

type publicReader struct {
	calls  atomic.Int64
	values []Message
}

func (r *publicReader) TeamMessages(_ context.Context, after, through int64, budget PageBudget) ([]Message, error) {
	r.calls.Add(1)
	var out []Message
	used := 0
	for _, value := range r.values {
		if value.Sequence > after && value.Sequence <= through {
			cost := value.ContextBytes()
			if !budget.Fits(len(out), used, cost) {
				break
			}
			out = append(out, value)
			used += cost
		}
	}
	return out, nil
}

func TestPublicHistoryRestoresOnceAndCopiesReceiverMetadata(t *testing.T) {
	reader := &publicReader{values: []Message{{Sequence: 10, From: "Hana", Text: strings.Repeat("x", 65536)}, {Sequence: 20, From: "user", Text: "current request"}}}
	var h History
	for range 8 {
		values, err := h.Read(context.Background(), reader, 0, 20, PublicPage(100))
		if err != nil || len(values) != 2 || values[1].Context.Content != "current request" {
			t.Fatalf("history: %d %v", len(values), err)
		}
		values[1].Context.Content = "private mutation"
	}
	if calls := reader.calls.Load(); calls != 1 {
		t.Fatalf("public bodies restored %d times", calls)
	}
}

func TestPublicHistoryPublicationAndEvictionKeepCoverage(t *testing.T) {
	reader := &publicReader{}
	var h History
	for i := int64(1); i <= 130; i++ {
		m := Message{Sequence: i * 2, From: "user", Text: "instruction"}
		reader.values = append(reader.values, m)
		h.Append(m, (i-1)*2)
	}
	if len(h.values) != 128 || h.after != 4 || h.bytes > historyBytes {
		t.Fatal("unbounded or incorrectly indexed cache")
	}
	page, err := h.Read(context.Background(), reader, 0, 260, PublicPage(100))
	if err != nil || len(page) != 100 || page[0].Message.Sequence != 2 || reader.calls.Load() != 1 {
		t.Fatal("evicted original was omitted")
	}
	// A new append after a partially restored older page cannot claim coverage of its gap.
	m := Message{Sequence: 262, From: "user", Text: "new"}
	reader.values = append(reader.values, m)
	h.Append(m, 260)
	page, err = h.Read(context.Background(), reader, 200, 262, PublicPage(100))
	if err != nil || len(page) != 31 || page[0].Message.Sequence != 202 {
		t.Fatalf("gap in history was hidden: %d %v", len(page), err)
	}
}

func TestPublicHistoryLargeBodyAndCancellation(t *testing.T) {
	reader := &publicReader{values: []Message{{Sequence: 1, From: "Hana", Text: strings.Repeat("x", historyBytes)}}}
	var h History
	values, err := h.Read(context.Background(), reader, 0, 1, PublicPage(100))
	if err != nil || len(values) != 1 || len(h.values) != 0 || h.bytes != 0 {
		t.Fatal("large accepted input was rejected or retained")
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := h.Read(ctx, reader, 0, 1, PublicPage(100)); !errors.Is(err, context.Canceled) {
		t.Fatal("cancelled read reached the cache or repository")
	}
}

func TestPublicHistoryChargesImagesAndDomainStorage(t *testing.T) {
	var h History
	for sequence := int64(1); sequence <= 3; sequence++ {
		image := media.Image{Data: make([]byte, 900<<10)}
		h.Append(Message{Sequence: sequence, From: "user", Text: "Inspect image", Images: []media.Image{image}}, sequence-1)
	}
	domain := source.WorkDomain{Domain: strings.Repeat("가", 128)}
	h.Append(Message{Sequence: 4, From: "Hana", Text: "Inspect", Declaration: true, Note: &runrecord.TeamNote{
		Work: &runrecord.TeamWork{Revision: 1, Domains: []source.WorkDomain{domain}},
	}}, 3)
	if len(h.values) != 3 || h.values[0].Message.Sequence != 2 {
		t.Fatalf("cache kept images beyond its budget: %d entries", len(h.values))
	}
	last := h.values[len(h.values)-1]
	if extra := last.retainedBytes() - len(last.Context.Content) - 512; extra < len(domain.Domain) {
		t.Fatalf("domain storage was omitted: %d bytes", extra)
	}
}
