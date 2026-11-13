// Purpose: Search beyond display prefixes without leaking masked values or trusting stale cursors.
// Pattern: Native API contract over real persisted message and reasoning records.
// Usage: go test ./internal/serverapi -run TestHistorySearch.
// Related: internal/serverapi/history_search.go, internal/serverapi/detail.go.
package serverapi

import (
	"strings"
	"testing"

	"github.com/keonho-kim/meca/internal/agent/model"
	"github.com/keonho-kim/meca/internal/agent/record"
	v1 "github.com/keonho-kim/meca/internal/serverapi/gen/mecav1"
)

func TestHistorySearchResumesLargeNarrativesAndMasksBeforeMatching(t *testing.T) {
	f := newFixture(t, generateFunc(idleGenerator), false)
	state, err := f.app.Select(t.Context(), "", true)
	if err != nil {
		t.Fatal(err)
	}
	secret := "never-search-this-key-123456"
	if err := f.api.Remember(secret); err != nil {
		t.Fatal(err)
	}
	text := strings.Repeat("가나다라 ", 70000) + "large-narrative-needle " + secret
	if err := f.store.AppendMessage(t.Context(), state.Session.ID, model.Message{Role: "assistant", Content: text}); err != nil {
		t.Fatal(err)
	}
	if err := f.store.RecordEvent(t.Context(), state.Session.ID, record.Event{Kind: "thinking", Thinking: "reasoning-needle"}); err != nil {
		t.Fatal(err)
	}
	wire := nativeTransport(t, f, "sse")
	request := &v1.HistorySearch{SessionId: state.Session.ID, Query: "LARGE-NARRATIVE-NEEDLE"}
	read := func() *v1.Reply {
		return nativeCommand(t, wire, &v1.Command{Operation: &v1.Command_SearchHistory{SearchHistory: request}})
	}
	first := read().GetHistoryMatches()
	if first == nil || len(first.Items) != 0 || first.Cursor == "" {
		t.Fatal("large search did not return bounded continuation")
	}
	request.Cursor = first.Cursor
	request.Query = "different-query"
	if read().GetError().GetCode() != "invalid_argument" {
		t.Fatal("query-bound cursor accepted another query")
	}
	request.Query = "LARGE-NARRATIVE-NEEDLE"
	found := false
	for range 5 {
		page := read().GetHistoryMatches()
		if page == nil {
			t.Fatal("search failed")
		}
		for _, hit := range page.Items {
			if strings.Contains(hit.Excerpt, secret) {
				t.Fatal("secret escaped masking")
			}
			if strings.Contains(hit.Excerpt, "large-narrative-needle") {
				found = true
			}
		}
		if page.Cursor == "" {
			break
		}
		request.Cursor = page.Cursor
	}
	if !found {
		t.Fatal("search missed content beyond display and first scan budgets")
	}
	request.Query, request.Cursor = "reasoning-needle", ""
	page := read().GetHistoryMatches()
	if page == nil || len(page.Items) != 1 || !page.Items[0].Thinking {
		t.Fatal("reasoning search lost its view")
	}
}
