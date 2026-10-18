// Purpose: Keep repeated delivery of one command distinct from actual repeated execution.
// Pattern: Pure execution-fact regression test.
// Usage: go test ./internal/agent/tool -run TestCommandFactsMerge.
// Related: internal/agent/tool/facts_merge.go, internal/agent/completion/facts.go.
package tool

import "testing"

func TestCommandFactsMergeRetainsSeparateExecutions(t *testing.T) {
	zero := 0
	first := CommandFact{ObservedRun: "run", CallID: "call", Revision: 1, Command: "test -f a", Status: "exited", ExitCode: &zero}
	second := first
	second.Revision = 2
	second.CallID = "second"
	facts := Facts{Commands: []CommandFact{first}}
	first.BeforeLaterChanges = true
	facts.Merge(Facts{Commands: []CommandFact{first, second}})
	if len(facts.Commands) != 2 || !facts.Commands[0].BeforeLaterChanges {
		t.Fatalf("resumed report duplicated old command: %+v", facts.Commands)
	}
}
