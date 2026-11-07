// Purpose: Verify page-local reuse preserves originals, projection, cancellation and integrity boundaries.
// Pattern: Storage hydration lifecycle regression tests.
// Usage: Run TestPagePayload and TestPageReference with -race.
// Related: internal/storage/sqlite/page_payloads.go, internal/storage/sqlite/page_references.go.
package sqlite

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"testing"

	runrecord "github.com/keonho-kim/meca/internal/agent/record"
)

func TestPagePayloadReleasesLastUseAndRechecksNextPage(t *testing.T) {
	s, err := newTestStore(t)
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	text := strings.Repeat("원본\n", 10000)
	for range 3 {
		if err := s.RecordEvent(ctx, s.Session().ID, runrecord.Event{Kind: "thinking", Thinking: text}); err != nil {
			t.Fatal(err)
		}
	}
	page, err := s.recordPage(ctx, "SELECT sequence,'event',body,refs,raw_bytes FROM tool_events ORDER BY sequence")
	if err != nil {
		t.Fatal(err)
	}
	p := pagePayloads{store: s, ctx: ctx}
	for i, record := range page {
		if err := p.include(i, p.store, record.refs); err != nil {
			t.Fatal(err)
		}
	}
	for i, record := range page {
		body, err := p.unpack(i, record, nil)
		var event runrecord.Event
		if err != nil || json.Unmarshal(body, &event) != nil || event.Thinking != text {
			t.Fatal("original changed", err)
		}
		if i == 0 && p.bytes == 0 || i == 2 && (p.bytes != 0 || len(p.bodies) != 0) {
			t.Fatal("payload reuse outlived its final record", i, p.bytes)
		}
	}
	if _, err := p.unpack(0, page[0], nil); err != nil {
		t.Fatal(err)
	}
	cancel()
	if _, err := p.unpack(1, page[1], nil); !errors.Is(err, context.Canceled) {
		t.Fatal("cached read ignored cancellation", err)
	}
	if _, err := s.db.Exec("UPDATE payloads SET body=x'00'"); err != nil {
		t.Fatal(err)
	}
	fresh := pagePayloads{store: s, ctx: context.Background()}
	if _, err := fresh.unpack(0, page[0], nil); err == nil {
		t.Fatal("another page reused an old authentication")
	}
}

func TestPagePayloadBudgetDoesNotRestrictLargeOriginals(t *testing.T) {
	s, err := newTestStore(t)
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	ctx := context.Background()
	text := strings.Repeat("x", pagePayloadBytes+1024)
	if err := s.RecordEvent(ctx, s.Session().ID, runrecord.Event{Kind: "thinking", Thinking: text}); err != nil {
		t.Fatal(err)
	}
	page, err := s.recordPage(ctx, "SELECT sequence,'event',body,refs,raw_bytes FROM tool_events")
	if err != nil || len(page) != 1 {
		t.Fatal(err)
	}
	p := pagePayloads{store: s, ctx: ctx}
	if err := p.include(1, p.store, page[0].refs); err != nil {
		t.Fatal(err)
	}
	body, err := p.unpack(0, page[0], nil)
	if err != nil || !strings.Contains(string(body), text) || p.bytes != 0 {
		t.Fatal("cache budget constrained an accepted original", err)
	}
}

func TestPageReferenceIndexStaysBoundedAndTracksExistingReferences(t *testing.T) {
	last, err := indexPayloadRefs(nil, nil, 0, []byte(`{"[\"output\"]":"kept"}`))
	if err != nil {
		t.Fatal(err)
	}
	refs := make(map[string]string, pageReferenceEntries*2)
	for i := range pageReferenceEntries * 2 {
		refs[fmt.Sprintf("field%d", i)] = fmt.Sprintf("payload%d", i)
	}
	body, err := json.Marshal(refs)
	if err != nil {
		t.Fatal(err)
	}
	last, err = indexPayloadRefs(last, nil, 1, body)
	if err != nil || len(last) != pageReferenceEntries {
		t.Fatal("reference index exceeded its page budget", len(last), err)
	}
	last, err = indexPayloadRefs(last, nil, 2, []byte(`{"[\"output\"]":"kept"}`))
	if err != nil || last[payloadKey{nil, "kept"}] != 2 || len(last) != pageReferenceEntries {
		t.Fatal("full index stopped updating a cached payload's last reference", err)
	}
}
