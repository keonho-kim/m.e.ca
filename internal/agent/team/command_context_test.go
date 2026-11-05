// Purpose: Preserve every active sibling command in compact shared context after individual completions.
// Pattern: Room observation regression test.
// Usage: go test ./internal/agent/team -run TestContextIncludesEveryActiveCommand.
// Related: internal/agent/team/command_status.go, internal/agent/team/work_context.go.
package team

import (
	"fmt"
	runrecord "github.com/keonho-kim/meca/internal/agent/record"
	toolapi "github.com/keonho-kim/meca/internal/agent/tool"
	"strings"
	"testing"
)

func TestContextIncludesEveryActiveCommand(t *testing.T) {
	r := testRoom(t)
	p := &Peer{ID: "Hana", room: r}
	for i := range 16 {
		id := fmt.Sprintf("cmd_%d", i)
		op := &toolapi.Operation{Name: "execute_command", Status: "tool_succeeded", Process: &toolapi.ProcessOutcome{CommandID: id, Status: "running"}}
		if err := p.Emit(runrecord.Event{Kind: "tool_succeeded", Operation: op}); err != nil {
			t.Fatal(err)
		}
	}
	other := &Peer{ID: "Ren", room: r}
	text := other.ContextObservations(nil)
	for i := range 16 {
		if !strings.Contains(text, fmt.Sprintf("cmd_%d", i)) {
			t.Fatalf("active sibling missing: %s", text)
		}
	}
	op := &toolapi.Operation{Name: "execute_command", Status: "tool_succeeded", Process: &toolapi.ProcessOutcome{CommandID: "cmd_0", Status: "exited"}}
	if err := p.Emit(runrecord.Event{Kind: "command_finished", Operation: op}); err != nil {
		t.Fatal(err)
	}
	text = other.ContextObservations(nil)
	for i := 1; i < 16; i++ {
		if !strings.Contains(text, fmt.Sprintf("cmd_%d", i)) {
			t.Fatalf("completed sibling erased another command: %s", text)
		}
	}
}

func TestCommandStatusReusesStableContributionSnapshot(t *testing.T) {
	r := testRoom(t)
	p := &Peer{ID: "Hana", room: r}
	if _, err := p.SetWork("API", []string{"/project/api"}, "contract", 0); err != nil {
		t.Fatal(err)
	}
	base := r.workSnapshot().view.base
	op := &toolapi.Operation{Name: "execute_command", Status: "tool_succeeded", Process: &toolapi.ProcessOutcome{CommandID: "cmd_own", Status: "running"}}
	if err := p.Emit(runrecord.Event{Kind: "tool_succeeded", Operation: op}); err != nil {
		t.Fatal(err)
	}
	if r.workSnapshot().view.base != base {
		t.Fatal("command status copied stable membership and contributions")
	}
}
