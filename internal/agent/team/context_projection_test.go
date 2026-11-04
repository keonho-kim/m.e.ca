// Purpose: Keep public context complete while removing duplicate repeated declarations and opinions.
// Pattern: Model-input projection regression tests.
// Usage: Team tests supply uncompacted history directly without invoking a provider.
// Related: internal/agent/team/context_projection.go, internal/agent/team/work_context.go.
package team

import (
	"encoding/json"
	model "github.com/keonho-kim/meca/internal/agent/model"
	"github.com/keonho-kim/meca/internal/agent/tool"
	"strings"
	"testing"
)

func TestContextProjectionRestoresDetailsOnlyWhenHistoryLacksThem(t *testing.T) {
	r := testRoom(t)
	a, b := &Peer{ID: "Hana", room: r}, &Peer{ID: "Ren", room: r}
	work, err := a.SetWork("Authentication", []string{"/project/auth"}, "Locked accounts return 423", 0)
	if err != nil {
		t.Fatal(err)
	}
	seq, err := a.Discuss(r.ctx, "Preserve existing client behavior", work.Sequence, nil)
	if err != nil {
		t.Fatal(err)
	}
	batch, err := b.Inbox(r.ctx)
	if err != nil {
		t.Fatal(err)
	}
	b.BeginDecision()
	text := b.ContextObservations(batch.Messages)
	if strings.Contains(text, "Locked accounts return 423") || strings.Contains(text, "Preserve existing client behavior") || !strings.Contains(text, "current work #") {
		t.Fatal("duplicate bodies in repeated context", text)
	}
	if repeated := b.ContextObservations(batch.Messages); repeated != text {
		t.Fatal("unchanged projection changed")
	}
	afterCompaction := b.ContextObservations(nil)
	if !strings.Contains(afterCompaction, "Locked accounts return 423") || !strings.Contains(afterCompaction, "Preserve existing client behavior") {
		t.Fatal("compaction lost current collaboration details", afterCompaction)
	}
	orphan := []model.Message{{Role: "tool", ToolName: "set_work", ToolCallID: "missing", Content: "{\"sequence\":1}"}}
	if text = b.ContextObservations(orphan); !strings.Contains(text, "Locked accounts return 423") {
		t.Fatal("an orphan result hid a declaration absent from context")
	}
	ownHistory := []model.Message{}
	for _, value := range []struct {
		name string
		seq  int64
	}{{"set_work", work.Sequence}, {"discuss", seq}} {
		raw, _ := json.Marshal(map[string]int64{"sequence": value.seq})
		ownHistory = append(ownHistory, model.Message{Role: "assistant", Calls: []tool.Call{{ID: value.name, Name: value.name, Arguments: json.RawMessage("{}")}}})
		ownHistory = append(ownHistory, model.Message{Role: "tool", ToolCallID: value.name, ToolName: value.name, Content: string(raw)})
	}
	if _, err = a.Inbox(r.ctx); err != nil {
		t.Fatal(err)
	}
	a.BeginDecision()
	if text = a.ContextObservations(ownHistory); strings.Contains(text, "Locked accounts return 423") || strings.Contains(text, "Preserve existing client behavior") {
		t.Fatal("own native results repeated their source bodies", text)
	}
}

func TestScopeBoundsRetainedTextAndComparesFullContent(t *testing.T) {
	r := testRoom(t)
	p := &Peer{ID: "Hana", room: r}
	summary, contract := strings.Repeat("s", 1<<20), strings.Repeat("c", 1<<20)
	first, err := p.SetWork(summary, []string{"/project/auth"}, contract, 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(first.Summary) > 4096 || len(first.Contract) > 4096 {
		t.Fatal("state retained full input")
	}
	same, err := p.SetWork(summary, []string{"/project/auth"}, contract, 0)
	if err != nil || same.Sequence != first.Sequence {
		t.Fatal("unchanged full content produced a publication", err)
	}
	changed, err := p.SetWork(summary, []string{"/project/auth"}, contract+"different tail", 0)
	if err != nil || changed.Revision != first.Revision+1 {
		t.Fatal("equal previews hid a full-content change", err)
	}
	notes, err := p.ReadMessages(r.ctx, 0)
	if err != nil || notes[len(notes)-1].Note.Work.Contract != contract+"different tail" {
		t.Fatal("full source was truncated", err)
	}
}
