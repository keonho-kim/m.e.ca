// Purpose: Preserve bounded state traversal and reset streamed offsets when credential masking changes.
// Pattern: Deterministic state and stream contract regression tests.
// Usage: go test -race ./internal/commands/server -run 'TestStatePages|TestMaskRevision'.
// Related: internal/commands/server/state_page.go, internal/commands/server/projection.go.
package server

import (
	"fmt"
	"github.com/keonho-kim/meca/internal/agent/record"
	"github.com/keonho-kim/meca/internal/commands/agentrequest"
	"github.com/keonho-kim/meca/internal/config/credentials"
	"strings"
	"testing"
)

func TestStatePagesTraverseEveryActiveTask(t *testing.T) {
	s, _ := projectionFixture()
	s.active = make(map[string]*request)
	for i := range 150 {
		id := fmt.Sprintf("task-%03d", i)
		s.active[id] = &request{task: Task{ID: id}}
	}
	seen := map[string]bool{}
	after := ""
	for {
		page, err := s.StatePage(t.Context(), "", "", after, 17)
		if err != nil || len(page.Tasks) > 17 || page.TotalTasks != 150 {
			t.Fatal(page, err)
		}
		for _, task := range page.Tasks {
			if seen[task.ID] {
				t.Fatal("duplicate page item", task.ID)
			}
			seen[task.ID] = true
		}
		if page.NextTask == "" {
			break
		}
		after = page.NextTask
	}
	if len(seen) != 150 {
		t.Fatal("state page omitted active tasks", len(seen))
	}
}

func TestMaskRevisionResetsDraftBeforeApplyingOffsets(t *testing.T) {
	s, r := projectionFixture()
	vault := &credentials.Vault{}
	r.conversation = &Conversation{Runtime: agentrequest.Runtime{Vault: vault}}
	s.emit(r, record.Event{Kind: "generation_started"})
	s.emit(r, record.Event{Kind: "stream", Text: "future-key" + strings.Repeat("a", 100)})
	cursor := s.State().Cursor
	generation := r.drafts[""].Generation
	if err := vault.Remember("future-key"); err != nil {
		t.Fatal(err)
	}
	s.emit(r, record.Event{Kind: "stream", Text: "future-key" + strings.Repeat("a", 200)})
	batch, err := s.Replay(cursor)
	if err != nil || len(batch.Events) != 2 || !batch.Events[0].Reset || batch.Events[1].Generation != generation+1 || batch.Events[1].Record.Text != "[redacted]"+strings.Repeat("a", 200) {
		t.Fatal("mask revision reused old source offsets", batch, err)
	}
}
