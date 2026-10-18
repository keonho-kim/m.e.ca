// Purpose: Stop repeated inspection and changing test-duration output without losing execution facts.
// Pattern: Native-loop integration regression.
// Usage: go test ./internal/agent/run -run TestCommandOutputVariationsCannotKeepLoopAlive.
// Related: internal/agent/run/semantic_progress.go, internal/agent/run/finish_response.go.
package run

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	toolapi "github.com/keonho-kim/v-code/internal/agent/tool"
)

func TestCommandOutputVariationsCannotKeepLoopAlive(t *testing.T) {
	g, e, j := &generator{}, &executor{}, &journal{}
	for i := range 20 {
		command, output := "cat main.py", "unchanged source"
		if i%2 == 0 {
			command, output = "pytest | tail", fmt.Sprintf("SyntaxError; error in 0.%02ds", i)
		}
		g.replies = append(g.replies, nativeCall("execute_command", map[string]any{"command": command}))
		body, _ := json.Marshal(map[string]string{"Stdout": output})
		e.outcomes = append(e.outcomes, toolapi.Result{Started: true, Content: body, Summary: output, Process: &toolapi.ProcessOutcome{Status: "exited"}})
	}
	r := fixture(g, e, j)
	result, err := r.Run(context.Background(), "Implement the server and pass its tests", approve, emit)
	if err != nil || result.Status != Stopped || e.executed > 8 {
		t.Fatalf("status=%s executed=%d err=%v", result.Status, e.executed, err)
	}
	var recorded strings.Builder
	for _, message := range j.messages {
		recorded.WriteString(message.Content)
	}
	if !strings.Contains(recorded.String(), "error in 0.00s") || !strings.Contains(recorded.String(), "error in 0.02s") {
		t.Fatal("stall detection discarded actual command outputs")
	}
}
