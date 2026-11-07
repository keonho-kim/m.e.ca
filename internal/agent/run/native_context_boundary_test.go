// Purpose: Keep native call/result pairs intact while admitting peer observations before the next decision.
// Pattern: Scripted context-boundary regression.
// Usage: Runtime tests publish a peer note during preparation without changing user instructions.
// Related: internal/agent/run/steering.go, internal/agent/context/pairing.go.
package run

import (
	"context"
	"github.com/keonho-kim/meca/internal/agent/team/source"
	"strings"
	"testing"

	model "github.com/keonho-kim/meca/internal/agent/model"
	"github.com/keonho-kim/meca/internal/agent/tool"
)

func TestPeerObservationDuringPreparationPreservesNativePairing(t *testing.T) {
	g := &generator{replies: []string{nativeCall("inspect_fixture", map[string]any{"path": "a"}), "Read the actual result and peer note"}}
	tools, j := &executor{}, &contextJournalFixture{}
	r := fixture(g, tools, &j.journal)
	r.Journal = j
	r.PollContext = func(context.Context) (ContextBatch, error) {
		if tools.prepared > 0 && j.cursor == 0 {
			return ContextBatch{Cursor: 7, Messages: []model.Message{{ID: "peer:Ren/7", Source: &source.Public{Author: "Ren", Kind: "team_message", Sequence: 7}, Role: "system", Content: "Public conversation — Ren\nI am reviewing the existing interface."}}}, nil
		}
		return ContextBatch{}, nil
	}
	if _, err := r.Run(context.Background(), "Inspect a", approve, emit); err != nil {
		t.Fatal(err)
	}
	paired, note := false, false
	for _, message := range g.requests[1].Messages {
		paired = paired || message.Role == "tool" && message.ToolName == "inspect_fixture"
		note = note || strings.Contains(message.Content, "I am reviewing the existing interface")
	}
	if !paired || !note || tools.executed != 1 {
		t.Fatal("observation admission expanded a valid native pair or lost context", paired, note, tools.executed)
	}
}

func TestPeerObservationBetweenWavesPreservesTheWholeNativeGroup(t *testing.T) {
	g := &parallelModel{calls: []tool.Call{
		batchCall("first", "inspect_fixture", map[string]any{"path": "first"}),
		batchCall("second", "inspect_fixture", map[string]any{"path": "second"}),
	}}
	tools, j := &executor{}, &contextJournalFixture{}
	r := fixture(&generator{}, tools, &j.journal)
	r.Journal, r.Generator = j, g
	r.PollContext = func(context.Context) (ContextBatch, error) {
		if tools.executed > 0 && j.cursor == 0 {
			return ContextBatch{Cursor: 9, Messages: []model.Message{{ID: "peer:Ren/9", Source: &source.Public{Author: "Ren", Kind: "team_message", Sequence: 9}, Role: "system", Content: "Public conversation — Ren\nThe interface review is complete."}}}, nil
		}
		return ContextBatch{}, nil
	}
	if _, err := r.Run(context.Background(), "Inspect both files", approve, emit); err != nil {
		t.Fatal(err)
	}
	paired, note := 0, false
	for _, message := range g.requests[1].Messages {
		if message.Role == "tool" {
			paired++
		}
		note = note || strings.Contains(message.Content, "The interface review is complete")
	}
	if paired != 2 || !note || tools.executed != 2 {
		t.Fatal("peer reception broke a valid batch or lost observations", paired, note, tools.executed)
	}
}
