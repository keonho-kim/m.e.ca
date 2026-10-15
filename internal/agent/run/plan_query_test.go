// Purpose: Verify Plan advertises command queries while blocking shell side effects before preparation.
// Pattern: Runtime boundary tests.
// Usage: go test ./internal/agent/run -run TestPlanCommand.
// Related: internal/agent/run/tool_policy.go, internal/agent/run/action.go, internal/policy/shell/query.go.
package run

import (
	"context"
	"testing"
)

func TestPlanCommandQueries(t *testing.T) {
	for _, tc := range []struct {
		command    string
		executions int
	}{
		{"find . -name '*.go'", 1}, {"ls -la", 1}, {"rg -n TODO .", 1},
		{"find . -delete", 0}, {"ls > marker", 0}, {"sh -c ls", 0},
	} {
		t.Run(tc.command, func(t *testing.T) {
			g, e := &generator{replies: []string{nativeCall("execute_command", map[string]any{"command": tc.command}), "Plan discussion"}}, &executor{}
			result, err := fixture(g, e, &journal{}).Run(context.Background(), "inspect before planning", true, approve, emit)
			if err != nil || result.Status != Answered || e.executed != tc.executions {
				t.Fatalf("%+v executions=%d %v", result, e.executed, err)
			}
			found := false
			for _, definition := range g.requests[0].Tools {
				if definition.Name == "execute_command" {
					found = true
				}
				if definition.Name == "write_file" {
					t.Fatal("Plan advertised mutation")
				}
			}
			if !found {
				t.Fatal("Plan query entry missing")
			}
		})
	}
}
