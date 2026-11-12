// Purpose: Keep input recall bounded before decoding large persisted content.
// Pattern: Temporary SQLite restoration contract test.
// Usage: go test ./internal/storage/sqlite -run TestRecentInputsByteBudget.
// Related: internal/storage/sqlite/input_history.go, internal/storage/history/inputs.go.
package sqlite

import (
	"fmt"
	"strings"
	"testing"

	"github.com/keonho-kim/meca/internal/agent/model"
	"github.com/keonho-kim/meca/internal/processmemory"
)

func TestRecentInputsByteBudgetKeepsWholeNewestEntries(t *testing.T) {
	store, err := newTestStore(t)
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	body := strings.Repeat("x", 512<<10)
	for index := range 34 {
		if err := store.AppendMessage(t.Context(), store.Session().ID, model.Message{Role: "user", Content: fmt.Sprintf("%03d%s", index, body)}); err != nil {
			t.Fatal(err)
		}
	}
	inputs, err := store.RecentInputs(t.Context(), store.Session().ID, 200)
	if err != nil {
		t.Fatal(err)
	}
	bytes := 0
	for _, text := range inputs {
		bytes += len(text)
		if len(text) != len(body)+3 {
			t.Fatal("recall returned a clipped editable input")
		}
	}
	if bytes > 16<<20 || len(inputs) == 0 || !strings.HasPrefix(inputs[len(inputs)-1], "033") {
		t.Fatalf("unbounded or stale recall: %d bytes", bytes)
	}
}

func TestRecentInputsSmallHistoryDoesNotReserveMaximum(t *testing.T) {
	s, err := newTestStore(t)
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	if err := s.AppendMessage(t.Context(), s.session.ID, model.Message{Role: "user", Content: "small editable input"}); err != nil {
		t.Fatal(err)
	}
	other := processmemory.NewPayloadLease()
	defer other.Close()
	if err := other.Resize(int(processmemory.PayloadBytes) - (4 << 20)); err != nil {
		t.Fatal(err)
	}
	inputs, err := s.RecentInputs(t.Context(), s.session.ID, 200)
	if err != nil || len(inputs) != 1 || inputs[0] != "small editable input" {
		t.Fatal(inputs, err)
	}
}

func TestRecentInputsPreservesEscapedPayload(t *testing.T) {
	s, err := newTestStore(t)
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	text := strings.Repeat("<>&🙂\t", 32<<10)
	if err := s.AppendMessage(t.Context(), s.session.ID, model.Message{Role: "user", Content: text}); err != nil {
		t.Fatal(err)
	}
	inputs, err := s.RecentInputs(t.Context(), s.session.ID, 200)
	if err != nil || len(inputs) != 1 || inputs[0] != text {
		t.Fatal("changed escaped input", len(inputs), err)
	}
}
