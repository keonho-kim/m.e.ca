//go:build !tui

// Purpose: Exercise two real parallel command executions through the embedded UI's full event and history path.
// Pattern: Deterministic native-call provider fixture.
// Usage: uiProvider routes PARALLEL_FIXTURE generations here.
// Related: cmd/meca/ui_provider_test.go, webui/e2e/tool-batches.spec.ts.
package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
)

func uiParallelReply(w http.ResponseWriter, completed bool) {
	delta, finish := map[string]any{}, "stop"
	if completed {
		delta["content"] = "Parallel fixtures inspected."
	} else {
		finish = "tool_calls"
		calls := []string{}
		for _, command := range []string{"printf parallel-first", "printf parallel-second"} {
			arguments, _ := json.Marshal(map[string]string{"reason": "Run fixture check", "command": command})
			calls = append(calls, "tools.execute_command("+string(arguments)+")")
		}
		code, _ := json.Marshal(map[string]string{"code": "return await Promise.all([" + strings.Join(calls, ",") + "]);"})
		delta["tool_calls"] = []any{map[string]any{"index": 0, "id": "parallel-code", "type": "function", "function": map[string]string{"name": "codemode", "arguments": string(code)}}}
	}
	body, _ := json.Marshal(map[string]any{"choices": []any{map[string]any{"delta": delta, "finish_reason": finish}}, "usage": map[string]int{"prompt_tokens": 32, "completion_tokens": 64}})
	fmt.Fprintf(w, "data: %s\n\ndata: [DONE]\n\n", body)
	_ = http.NewResponseController(w).Flush()
}
