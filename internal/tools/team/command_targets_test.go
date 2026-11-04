// Purpose: Release shell target references at terminal delivery without exposing them in stored operations.
// Pattern: Command metadata ownership regression test.
// Usage: Team command tests cover running, completion and unrelated command ownership.
// Related: internal/tools/team/command_targets.go, internal/agent/tool/operation.go.
package team

import (
	"encoding/json"
	"strings"
	"testing"

	toolapi "github.com/keonho-kim/meca/internal/agent/tool"
)

func TestCommandTargetsSurviveUntilTerminalDeliveryOnly(t *testing.T) {
	e := &Executor{}
	e.trackCommandPaths("one", []string{"/project/first"})
	e.trackCommandPaths("two", []string{"/project/second"})
	if len(e.CommandTargets("one", false)) != 1 {
		t.Fatal("running targets disappeared")
	}
	paths := e.CommandTargets("one", true)
	if len(paths) != 1 || len(e.CommandTargets("one", false)) != 0 || len(e.CommandTargets("two", false)) != 1 {
		t.Fatal("terminal cleanup crossed command ownership")
	}
	e.CommandTargets("two", true)
	if e.commandPaths != nil {
		t.Fatal("completed command targets retained")
	}
	body, err := json.Marshal(toolapi.Operation{Name: "execute_command", AffectedPaths: paths})
	if err != nil || strings.Contains(string(body), "/project/first") {
		t.Fatal("transient targets leaked into persistent metadata", err)
	}
}
