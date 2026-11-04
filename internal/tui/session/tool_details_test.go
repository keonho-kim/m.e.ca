// Purpose: Verify compact tool records hydrate only on expansion and discard detail memory on collapse.
// Pattern: Temporary-store terminal regression tests.
// Usage: go test ./internal/tui/session -run 'TestLazyTool|TestReadme|TestConversationResumes'.
// Related: internal/tui/session/tool_details.go, internal/tui/viewport/details.go.
package session

import (
	runrecord "github.com/keonho-kim/meca/internal/agent/record"
	toolapi "github.com/keonho-kim/meca/internal/agent/tool"
	"strings"
	"testing"
	"time"
)

func settleVisibleDetails(t *testing.T, a *app) {
	t.Helper()
	deadline := time.NewTimer(5 * time.Second)
	defer deadline.Stop()
	for {
		a.render()
		if !a.chat.details.loading {
			return
		}
		select {
		case n := <-a.events:
			a.receive(n)
		case <-deadline.C:
			t.Fatal("tool detail worker did not finish")
		}
	}
}
func TestLazyToolDetailsExpandAndRelease(t *testing.T) {
	a := fixture(t)
	event := runrecord.Event{Kind: "tool_succeeded", CallID: "change", Text: "File changed", Call: &toolapi.Call{ID: "change", Name: "edit_file", Arguments: []byte(`{"path":"app.go"}`)}, Output: []byte(`{"changed":true}`), ChangePreview: "--- app.go\n+++ app.go\n@@ -1 +1 @@\n-old\n+new\n"}
	if err := a.opts.Store.RecordEvent(a.ctx, a.opts.Session.ID, event); err != nil {
		t.Fatal(err)
	}
	a.chat.output.Collapsed = true
	if err := a.loadHistory(); err != nil {
		t.Fatal(err)
	}
	a.render()
	block := a.chat.output.Blocks[0]
	if block.DetailRef == nil || block.Details != "" || block.Changes != nil || a.chat.details.loading {
		t.Fatal("collapsed result hydrated original details")
	}
	a.toggleToolDetails()
	settleVisibleDetails(t, a)
	block = a.chat.output.Blocks[0]
	if !strings.Contains(block.Text, "new") || block.Changes == nil || block.Details == "" {
		t.Fatalf("detail/color classification missing: %+v", block)
	}
	a.toggleToolDetails()
	a.render()
	block = a.chat.output.Blocks[0]
	if block.Details != "" || block.Changes != nil || block.DetailRef == nil || block.Text != block.Summary {
		t.Fatal("collapse retained original detail body")
	}
	a.toggleToolDetails()
	settleVisibleDetails(t, a)
	if !strings.Contains(a.chat.output.Blocks[0].Text, "new") {
		t.Fatal("re-expansion failed")
	}
}
